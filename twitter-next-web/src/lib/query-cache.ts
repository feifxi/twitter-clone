import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-keys';
import type { TweetResponse, UserResponse } from '@/types';

/*
 * Optimistic updates for entities that are cached in many places. One tweet can
 * sit in a feed, a profile tab, search results and its detail page (and twice
 * in a timeline when someone retweeted it), in caches of different shapes. These
 * helpers update every copy in one pass.
 */

type Updater<T> = (item: T) => T | null;

const isTweet = (x: unknown): x is TweetResponse =>
  typeof x === 'object' && x !== null && 'likeCount' in x && 'retweetCount' in x;

const isUser = (x: unknown): x is UserResponse =>
  typeof x === 'object' && x !== null && 'followersCount' in x && 'username' in x;

/**
 * Maps the matching items of a cached value, whatever its shape: one entity, an
 * array, a page (`{ items }`) or infinite pages (`{ pages }`). Returning null
 * removes the item from lists; a single cached entity is left as is.
 */
function mapCached<T>(data: unknown, isItem: (x: unknown) => x is T, fn: Updater<T>): unknown {
  if (data === null || typeof data !== 'object') return data;
  if (Array.isArray(data)) {
    return data.flatMap((item) => {
      if (!isItem(item)) return [mapCached(item, isItem, fn)];
      const next = fn(item);
      return next === null ? [] : [next];
    });
  }
  if ('pages' in data && Array.isArray(data.pages)) {
    return { ...data, pages: data.pages.map((page: unknown) => mapCached(page, isItem, fn)) };
  }
  if ('items' in data && Array.isArray(data.items)) {
    return { ...data, items: mapCached(data.items, isItem, fn) };
  }
  if (isItem(data)) return fn(data) ?? data;
  return data;
}

const TWEET_ROOTS: readonly QueryKey[] = [queryKeys.feeds.all, queryKeys.tweets.all, queryKeys.users.all, queryKeys.search.all];
const USER_ROOTS: readonly QueryKey[] = [...TWEET_ROOTS, queryKeys.auth.me, queryKeys.discovery.all];

function setEverywhere(queryClient: QueryClient, roots: readonly QueryKey[], map: (data: unknown) => unknown) {
  for (const queryKey of roots) {
    queryClient.setQueriesData({ queryKey }, map);
  }
}

/** Updates every cached copy of a tweet. Unchanged caches keep their identity (structural sharing). */
export function updateTweet(queryClient: QueryClient, tweetId: number, fn: Updater<TweetResponse>) {
  setEverywhere(queryClient, TWEET_ROOTS, (data) =>
    mapCached(data, isTweet, (t) => (t.id === tweetId ? fn(t) : t)),
  );
}

/** Updates every cached copy of a user, including authors embedded in tweets. */
export function updateUser(queryClient: QueryClient, userId: number, fn: (u: UserResponse) => UserResponse) {
  const apply = <U extends UserResponse | null>(u: U): U => (u && u.id === userId ? (fn(u) as U) : u);
  setEverywhere(queryClient, USER_ROOTS, (data) => {
    const users = mapCached(data, isUser, (u) => apply(u));
    return mapCached(users, isTweet, (t) => ({ ...t, user: apply(t.user), retweetedBy: apply(t.retweetedBy) }));
  });
}

/** Snapshot of every cache a tweet or user can live in, to roll an optimistic update back. */
export function snapshotEntities(queryClient: QueryClient): () => void {
  const saved = USER_ROOTS.flatMap((queryKey) => queryClient.getQueriesData({ queryKey }));
  return () => saved.forEach(([key, data]) => queryClient.setQueryData(key, data));
}

/** Refetches everything an optimistic tweet/user update may have touched. */
export function invalidateEntities(queryClient: QueryClient) {
  for (const queryKey of USER_ROOTS) {
    queryClient.invalidateQueries({ queryKey });
  }
}
