import { QueryClient, type InfiniteData } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';
import { snapshotEntities, updateTweet, updateUser } from './query-cache';
import { queryKeys } from './query-keys';
import type { PageResponse, TweetResponse, UserResponse } from '@/types';

const user = (id: number, extra: Partial<UserResponse> = {}): UserResponse => ({
  id,
  username: `u${id}`,
  displayName: null,
  avatarUrl: null,
  bannerUrl: null,
  bio: null,
  location: null,
  website: null,
  followersCount: 10,
  followingCount: 5,
  isFollowing: false,
  createdAt: '2026-01-01T00:00:00Z',
  ...extra,
});

const tweet = (id: number, extra: Partial<TweetResponse> = {}): TweetResponse => ({
  id,
  content: `tweet ${id}`,
  mediaType: null,
  mediaUrl: null,
  user: user(100),
  replyCount: 0,
  likeCount: 1,
  retweetCount: 0,
  isLiked: false,
  isRetweeted: false,
  createdAt: '2026-01-01T00:00:00Z',
  retweetedBy: null,
  retweetedAt: null,
  ...extra,
});

const page = <T,>(items: T[]): PageResponse<T> => ({ items, hasNext: false });
const pages = <T,>(...p: T[][]): InfiniteData<PageResponse<T>> => ({ pages: p.map(page), pageParams: p.map(() => null) });

describe('updateTweet', () => {
  it('updates every cached copy, whatever the cache shape', () => {
    const qc = new QueryClient();
    qc.setQueryData(queryKeys.feeds.global, pages([tweet(1), tweet(2)], [tweet(1, { retweetedBy: user(7) })]));
    qc.setQueryData(queryKeys.search.tweets('x'), page([tweet(1)]));
    qc.setQueryData(queryKeys.tweets.detail(1), tweet(1));
    qc.setQueryData(queryKeys.users.timeline(100, 'posts'), pages([tweet(1)]));

    updateTweet(qc, 1, (t) => ({ ...t, isLiked: true, likeCount: t.likeCount + 1 }));

    const feed = qc.getQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKeys.feeds.global)!;
    expect(feed.pages[0].items.map((t) => [t.id, t.isLiked])).toEqual([[1, true], [2, false]]);
    expect(feed.pages[1].items[0].likeCount).toBe(2); // the retweeted copy too
    expect(qc.getQueryData<PageResponse<TweetResponse>>(queryKeys.search.tweets('x'))!.items[0].isLiked).toBe(true);
    expect(qc.getQueryData<TweetResponse>(queryKeys.tweets.detail(1))!.isLiked).toBe(true);
    expect(
      qc.getQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKeys.users.timeline(100, 'posts'))!.pages[0].items[0].isLiked,
    ).toBe(true);
  });

  it('removes list items when the updater returns null, but keeps a cached detail', () => {
    const qc = new QueryClient();
    qc.setQueryData(queryKeys.feeds.global, pages([tweet(1), tweet(2)]));
    qc.setQueryData(queryKeys.tweets.detail(1), tweet(1));

    updateTweet(qc, 1, () => null);

    expect(qc.getQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKeys.feeds.global)!.pages[0].items.map((t) => t.id)).toEqual([2]);
    expect(qc.getQueryData<TweetResponse>(queryKeys.tweets.detail(1))!.id).toBe(1);
  });

  it('leaves caches that hold no matching tweet untouched', () => {
    const qc = new QueryClient();
    const other = pages([tweet(2)]);
    qc.setQueryData(queryKeys.feeds.following, other);
    qc.setQueryData(queryKeys.discovery.users(5), page([user(1)]));

    updateTweet(qc, 1, (t) => ({ ...t, isLiked: true }));

    expect(qc.getQueryData(queryKeys.feeds.following)).toBe(other); // same reference: no re-render
  });
});

describe('updateUser', () => {
  it('updates profiles, user lists, the current user and tweet authors', () => {
    const qc = new QueryClient();
    qc.setQueryData(queryKeys.users.byUsername('u100'), user(100));
    qc.setQueryData(queryKeys.discovery.users(5), page([user(100), user(101)]));
    qc.setQueryData(queryKeys.search.users('u', 10), page([user(100)]));
    qc.setQueryData(queryKeys.auth.me, { ...user(100), email: 'me@example.com', onboarded: true });
    qc.setQueryData(queryKeys.feeds.global, pages([tweet(1), tweet(2, { user: user(101), retweetedBy: user(100) })]));

    updateUser(qc, 100, (u) => ({ ...u, isFollowing: true, followersCount: u.followersCount + 1 }));

    expect(qc.getQueryData<UserResponse>(queryKeys.users.byUsername('u100'))!.followersCount).toBe(11);
    expect(qc.getQueryData<PageResponse<UserResponse>>(queryKeys.discovery.users(5))!.items.map((u) => u.isFollowing)).toEqual([true, false]);
    expect(qc.getQueryData<PageResponse<UserResponse>>(queryKeys.search.users('u', 10))!.items[0].isFollowing).toBe(true);
    const me = qc.getQueryData<UserResponse & { email: string }>(queryKeys.auth.me)!;
    expect([me.isFollowing, me.email]).toEqual([true, 'me@example.com']); // private fields survive
    const [t1, t2] = qc.getQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKeys.feeds.global)!.pages[0].items;
    expect(t1.user.isFollowing).toBe(true);
    expect([t2.user.isFollowing, t2.retweetedBy?.isFollowing]).toEqual([false, true]);
  });
});

describe('snapshotEntities', () => {
  it('rolls every touched cache back', () => {
    const qc = new QueryClient();
    qc.setQueryData(queryKeys.feeds.global, pages([tweet(1)]));
    qc.setQueryData(queryKeys.tweets.detail(1), tweet(1));
    const rollback = snapshotEntities(qc);

    updateTweet(qc, 1, () => null);
    updateTweet(qc, 1, (t) => ({ ...t, likeCount: 99 }));
    rollback();

    expect(qc.getQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKeys.feeds.global)!.pages[0].items).toHaveLength(1);
    expect(qc.getQueryData<TweetResponse>(queryKeys.tweets.detail(1))!.likeCount).toBe(1);
  });
});
