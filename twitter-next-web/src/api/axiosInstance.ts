import axios, { type AxiosError } from 'axios';
import { useAuthStore } from '@/store/useAuthStore';
import { useUIStore } from '@/store/useUIStore';
import type { AuthResponse, ErrorResponse, FieldErrors } from '@/types';

const baseURL = process.env.NEXT_PUBLIC_API_URL;

export const axiosInstance = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
});

// Request: inject access token from Zustand store
axiosInstance.interceptors.request.use(
  (config) => {
    const token = useAuthStore.getState().accessToken;
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response: on 401, try refresh then retry; map validation errors for forms
axiosInstance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ErrorResponse>) => {
    const originalRequest = error.config;

    if (!originalRequest) {
      return Promise.reject(error);
    }

    // 401: try refresh (refresh endpoint must not send Authorization)
    if (error.response?.status === 401 && !(originalRequest as { _retry?: boolean })._retry) {
      (originalRequest as { _retry?: boolean })._retry = true;

      try {
        const refreshToken = useAuthStore.getState().refreshToken;
        if (!refreshToken) {
          throw new Error('No refresh token available');
        }

        const { data } = await axios.post<AuthResponse>(
          `${baseURL}/auth/refresh`,
          { refreshToken }
        );
        useAuthStore.getState().setAuth(data.accessToken, data.refreshToken, data.user);
        originalRequest.headers.Authorization = `Bearer ${data.accessToken}`;
        return axiosInstance(originalRequest);
      } catch {
        useAuthStore.getState().logout();
        // Trigger generic Sign In modal instead of hard redirect
        useUIStore.getState().openSignInModal();
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  }
);

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
