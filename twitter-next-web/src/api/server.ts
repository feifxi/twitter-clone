import 'server-only';
import { cookies } from 'next/headers';
import type { CurrentUserResponse } from '@/types';

const baseURL = process.env.NEXT_PUBLIC_API_URL;

export const SESSION_COOKIES = ['access_token', 'refresh_token'] as const;

/** The session cookies of the incoming request, as a Cookie header for the API. */
async function sessionCookieHeader(): Promise<string> {
  const jar = await cookies();
  return SESSION_COOKIES.flatMap((name) => {
    const value = jar.get(name)?.value;
    return value ? [`${name}=${value}`] : [];
  }).join('; ');
}

/**
 * GET from the API while rendering on the server, as the visitor (their session
 * cookies are forwarded). Returns null for 401/404 so pages can render the
 * signed-out or not-found state; other failures throw.
 */
export async function serverFetch<T>(path: string): Promise<T | null> {
  const cookie = await sessionCookieHeader();
  const response = await fetch(`${baseURL}${path}`, {
    headers: cookie ? { cookie } : undefined,
    cache: 'no-store',
  });
  if (response.status === 401 || response.status === 404) return null;
  if (!response.ok) throw new Error(`GET ${path}: ${response.status}`);
  return (await response.json()) as T;
}

/**
 * The signed-in user for the first render: null when signed out, undefined
 * when it can't be told (API unreachable), in which case the browser asks itself.
 */
export async function getCurrentUser(): Promise<CurrentUserResponse | null | undefined> {
  const jar = await cookies();
  if (!SESSION_COOKIES.some((name) => jar.has(name))) return null;
  try {
    return (await serverFetch<CurrentUserResponse>('/auth/me')) ?? undefined;
  } catch {
    return undefined;
  }
}
