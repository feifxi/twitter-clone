'use client';

import { useInfiniteQuery } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import type { PageResponse, TweetResponse } from '@/types';

const PAGE_SIZE = 20;

export type FeedKind = 'global' | 'following';

export function feedQueryKey(kind: FeedKind) {
  return ['feeds', kind] as const;
}

async function fetchFeed(
  kind: FeedKind,
  cursor: string | null
): Promise<PageResponse<TweetResponse>> {
  const path = kind === 'global' ? '/feeds/global' : '/feeds/following';
  const { data } = await axiosInstance.get<PageResponse<TweetResponse>>(path, {
    params: { cursor: cursor ?? undefined, size: PAGE_SIZE },
  });
  return data;
}

export function useGlobalFeed(enabled = true) {
  return useInfiniteQuery({
    queryKey: feedQueryKey('global'),
    queryFn: ({ pageParam }) => fetchFeed('global', pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.hasNext ? lastPage.nextCursor : undefined,
    enabled,
  });
}

export function useFollowingFeed(enabled = true) {
  return useInfiniteQuery({
    queryKey: feedQueryKey('following'),
    queryFn: ({ pageParam }) => fetchFeed('following', pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.hasNext ? lastPage.nextCursor : undefined,
    enabled,
  });
}
