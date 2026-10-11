import 'server-only';
import { notFound } from 'next/navigation';
import { dehydrate, QueryClient } from '@tanstack/react-query';
import { getUserByUsername, prefetchFirstPage } from '@/api/server';
import { PAGE_SIZE, queryKeys, type ProfileTab } from '@/lib/query-keys';

type Params = Promise<{ username: string }>;

/** The profile's user, or the not-found page. Cached per request (see getUserByUsername). */
export async function requireProfileUser(params: Params) {
  const username = decodeURIComponent((await params).username);
  const user = await getUserByUsername(username);
  if (!user) notFound();
  return { username, user };
}

/** Dehydrated first page of one of the profile's lists, for a HydrationBoundary. */
export async function prefetchProfileList(params: Params, list: ProfileTab | 'followers' | 'following') {
  const { username, user } = await requireProfileUser(params);
  const queryClient = new QueryClient();
  queryClient.setQueryData(queryKeys.users.byUsername(username), user);
  const key =
    list === 'followers' || list === 'following' ? queryKeys.users[list](user.id) : queryKeys.users.timeline(user.id, list);
  await prefetchFirstPage(queryClient, key, `/users/${user.id}/${list}?size=${PAGE_SIZE}`);
  return { username, state: dehydrate(queryClient) };
}
