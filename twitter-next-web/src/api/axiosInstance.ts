import axios, { isAxiosError, type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import type { ErrorResponse, FieldErrors } from '@/types';

const baseURL = process.env.NEXT_PUBLIC_API_URL;

/**
 * The session lives in httpOnly cookies set by the API: the browser attaches
 * them (withCredentials) and page scripts never see the tokens.
 */
export const axiosInstance = axios.create({
  baseURL,
  withCredentials: true,
  headers: { 'Content-Type': 'application/json' },
});

let refreshing: Promise<boolean> | null = null;

/**
 * Renews the session cookies. Concurrent callers share one request, because a
 * refresh token works only once.
 */
export function refreshSession(): Promise<boolean> {
  refreshing ??= axios
    .post(`${baseURL}/auth/refresh`, null, { withCredentials: true })
    .then(
      () => true,
      () => false,
    )
    .finally(() => {
      refreshing = null;
    });
  return refreshing;
}

type SessionLostListener = (event: { request: InternalAxiosRequestConfig }) => void;
const sessionLostListeners = new Set<SessionLostListener>();

/** Called when a request fails for lack of a valid session (after refreshing didn't help). */
export function onSessionLost(listener: SessionLostListener): () => void {
  sessionLostListeners.add(listener);
  return () => sessionLostListeners.delete(listener);
}

type RetriableRequest = InternalAxiosRequestConfig & { _retried?: boolean };

axiosInstance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ErrorResponse>) => {
    const request = error.config as RetriableRequest | undefined;
    if (error.response?.status !== 401 || !request) {
      return Promise.reject(error);
    }

    // TOKEN_EXPIRED: we sent credentials but they're stale. Renew and retry once.
    // Retry even if this refresh failed: another tab may have just renewed the cookies.
    const expired = error.response.data?.code === 'TOKEN_EXPIRED';
    if (expired && !request._retried) {
      request._retried = true;
      await refreshSession();
      return axiosInstance(request);
    }

    if (expired) {
      // The session is gone for good. Drop its cookies so public pages work anonymously again.
      await axios.post(`${baseURL}/auth/logout`, null, { withCredentials: true }).catch(() => undefined);
    }
    sessionLostListeners.forEach((listener) => listener({ request }));
    return Promise.reject(error);
  },
);

/**
 * fetch() with the session cookies, renewing them once on TOKEN_EXPIRED. For
 * streaming responses, which axios can't read incrementally in the browser.
 */
export async function fetchWithSession(path: string, init: RequestInit = {}): Promise<Response> {
  const send = () => fetch(`${baseURL}${path}`, { ...init, credentials: 'include' });
  const response = await send();
  if (response.status !== 401) return response;

  const body = (await response.clone().json().catch(() => null)) as ErrorResponse | null;
  if (body?.code !== 'TOKEN_EXPIRED') return response;
  await refreshSession();
  return send();
}

export function isUnauthorized(error: unknown): boolean {
  return isAxiosError(error) && error.response?.status === 401;
}

/**
 * Extract backend validation errors into a field -> message map
 * for easy binding to form inputs (e.g. errors.content, errors.email).
 */
export function getFieldErrors(error: unknown): FieldErrors {
  const err = error as AxiosError<ErrorResponse>;
  const data = err.response?.data;
  const list = data?.details;
  if (!Array.isArray(list)) return {};
  return list.reduce<FieldErrors>((acc, { field, message }) => {
    acc[field] = message;
    return acc;
  }, {});
}

/**
 * Get the general error message from the backend (e.g. for toasts).
 */
export function getErrorMessage(error: unknown): string {
  const err = error as AxiosError<ErrorResponse>;
  const data = err.response?.data;
  return data?.message ?? err.message ?? 'Something went wrong';
}
