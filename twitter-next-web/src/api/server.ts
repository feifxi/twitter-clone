import 'server-only';
import { cache } from 'react';
import { cookies, headers } from 'next/headers';
import type { QueryClient, QueryKey } from '@tanstack/react-query';
import type { CurrentUserResponse, PageResponse, TweetResponse, UserResponse } from '@/types';

// On the server, call the API directly (in production: over the Docker network,
// API_INTERNAL_URL=http://api:8080/api/v1); browsers use NEXT_PUBLIC_API_URL ("/api/v1").
const baseURL = process.env.API_INTERNAL_URL ?? process.env.NEXT_PUBLIC_API_URL;

export const SESSION_COOKIES = ['access_token', 'refresh_token'] as const;

/** The session cookies of the incoming request, as a Cookie header for the API. */
async function sessionCookieHeader(): Promise<string> {
  const jar = await cookies();
  return SESSION_COOKIES.flatMap((name) => {
    const value = jar.get(name)?.value;
    return value ? [`${name}=${value}`] : [];
  }).join('; ');
}

async function get(path: string, cookie: string): Promise<Response> {
  // Pass the visitor's IP along (Caddy sets it), so the API rate-limits per visitor
  // rather than treating every server-rendered request as coming from this server.
  const forwardedFor = (await headers()).get('x-forwarded-for');
  return fetch(`${baseURL}${path}`, {
    headers: { ...(cookie ? { cookie } : {}), ...(forwardedFor ? { 'x-forwarded-for': forwardedFor } : {}) },
    cache: 'no-store',
  });
}

/**
 * GET from the API while rendering on the server, as the visitor (their session
 * cookies are forwarded). Returns null on 404. If the session turns out to be
 * stale (proxy.ts couldn't renew it), the page renders anonymously and the
 * browser renews the session and refetches. Other failures throw.
 */
export async function serverFetch<T>(path: string): Promise<T | null> {
  const cookie = await sessionCookieHeader();
  let response = await get(path, cookie);
  if (response.status === 401 && cookie) response = await get(path, '');
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`GET ${path}: ${response.status}`);
  return (await response.json()) as T;
}

/**
 * The signed-in user for the first render: null when signed out, undefined
 * when it can't be told (stale session, API unreachable), in which case the
 * browser asks itself. Cached per request (root and section layouts share it).
 */
export const getCurrentUser = cache(async (): Promise<CurrentUserResponse | null | undefined> => {
  const jar = await cookies();
  if (!SESSION_COOKIES.some((name) => jar.has(name))) return null;
  try {
    const response = await get('/auth/me', await sessionCookieHeader());
    if (!response.ok) return undefined;
    return (await response.json()) as CurrentUserResponse;
  } catch {
    return undefined;
  }
});

// cache(): generateMetadata and the page share one request per render.
export const getTweet = cache((id: number) => serverFetch<TweetResponse>(`/tweets/${id}`));
export const getUserByUsername = cache((username: string) =>
  serverFetch<UserResponse>(`/users/by-username/${encodeURIComponent(username)}`),
);

/**
 * Loads the first page of a list into an infinite query, as useInfiniteQuery
 * would. Best effort: on failure the browser fetches it instead.
 */
export async function prefetchFirstPage(queryClient: QueryClient, queryKey: QueryKey, path: string) {
  try {
    const page = await serverFetch<PageResponse<unknown>>(path);
    if (page) queryClient.setQueryData(queryKey, { pages: [page], pageParams: [null] });
  } catch {
    // Rendered without it; the client query loads it.
  }
}
