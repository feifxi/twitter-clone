import type { QueryClient, QueryKey } from '@tanstack/react-query';
import { queryKeys } from '@/lib/query-keys';
import type { FeedItemResponse, TweetResponse, UserResponse } from '@/types';

/*
 * Optimistic updates for entities that are cached in many places. One tweet can
 * sit in a feed, a profile tab, search results and its detail page (and twice
 * in a timeline when someone retweeted it), in caches of different shapes:
 * plain tweets, or feed items that wrap one. These helpers update every copy in
 * one pass.
 */

type Updater<T> = (item: T) => T | null;

const isTweet = (x: unknown): x is TweetResponse =>
  typeof x === 'object' && x !== null && 'likeCount' in x && 'retweetCount' in x;

const isUser = (x: unknown): x is UserResponse =>
  typeof x === 'object' && x !== null && 'followersCount' in x && 'username' in x;

const isFeedItem = (x: unknown): x is FeedItemResponse =>
  typeof x === 'object' && x !== null && 'kind' in x && 'tweet' in x && isTweet(x.tweet);

/** A timeline entry or a plain tweet (search results), as a feed item. */
export const asFeedItem = (x: FeedItemResponse | TweetResponse): FeedItemResponse =>
  isFeedItem(x) ? x : { kind: 'tweet', tweet: x, retweet: null };

const REMOVED = Symbol('removed');

function mapDeep<T>(data: unknown, isItem: (x: unknown) => x is T, fn: Updater<T>): unknown {
  if (data === null || typeof data !== 'object') return data;
  if (isItem(data)) return fn(data) ?? REMOVED;
  if (Array.isArray(data)) return data.map((item) => mapDeep(item, isItem, fn)).filter((item) => item !== REMOVED);
  if ('pages' in data && Array.isArray(data.pages)) {
    return { ...data, pages: data.pages.map((page: unknown) => mapDeep(page, isItem, fn)) };
  }
  if ('items' in data && Array.isArray(data.items)) {
    return { ...data, items: mapDeep(data.items, isItem, fn) };
  }
  // A feed item goes (or stays) with its tweet.
  if (isFeedItem(data)) {
    const tweet = mapDeep(data.tweet, isItem, fn);
    if (tweet === REMOVED) return REMOVED;
    return tweet === data.tweet ? data : { ...data, tweet };
  }
  return data;
}

/**
 * Maps the matching items of a cached value, whatever its shape: one entity, an
 * array, a page (`{ items }`), infinite pages (`{ pages }`), or feed items that
 * wrap a tweet. Returning null removes the item from lists (with the feed items
 * wrapping it); a single cached entity is left as is.
 */
function mapCached<T>(data: unknown, isItem: (x: unknown) => x is T, fn: Updater<T>): unknown {
  const next = mapDeep(data, isItem, fn);
  return next === REMOVED ? data : next;
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

/** Removes timeline entries (e.g. my retweet of a tweet, once I undo it). */
export function removeFeedItems(queryClient: QueryClient, matches: (item: FeedItemResponse) => boolean) {
  setEverywhere(queryClient, TWEET_ROOTS, (data) => mapCached(data, isFeedItem, (item) => (matches(item) ? null : item)));
}

/** Updates every cached copy of a user, including tweet authors and retweeters. */
export function updateUser(queryClient: QueryClient, userId: number, fn: (u: UserResponse) => UserResponse) {
  const apply = (u: UserResponse): UserResponse => (u.id === userId ? fn(u) : u);
  setEverywhere(queryClient, USER_ROOTS, (data) => {
    const users = mapCached(data, isUser, apply);
    const tweets = mapCached(users, isTweet, (t) => ({ ...t, user: apply(t.user) }));
    return mapCached(tweets, isFeedItem, (item) =>
      item.retweet ? { ...item, retweet: { ...item.retweet, retweeter: apply(item.retweet.retweeter) } } : item,
    );
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
