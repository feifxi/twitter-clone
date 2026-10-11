'use client';

import type { InfiniteData, UseInfiniteQueryResult } from '@tanstack/react-query';
import { EmptyState, ErrorState, type EmptyCopy } from '@/components/ListStates';
import { Tweet, TweetSkeleton } from '@/components/Tweet';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import { asFeedItem } from '@/lib/query-cache';
import type { FeedItemResponse, PageResponse, TweetResponse } from '@/types';

interface FeedListProps {
  /** Timelines return feed items; other tweet lists (search) return plain tweets. */
  query: UseInfiniteQueryResult<InfiniteData<PageResponse<FeedItemResponse>> | InfiniteData<PageResponse<TweetResponse>>>;
  empty: EmptyCopy;
}

/** An infinitely scrolling list of tweets with loading, error and empty states. */
export function FeedList({ query, empty }: FeedListProps) {
  const sentinel = useInfiniteScroll(query);
  const items = query.data?.pages.flatMap((page) => page.items.map(asFeedItem)) ?? [];

  if (query.isPending) {
    return Array.from({ length: 5 }, (_, i) => <TweetSkeleton key={i} />);
  }
  if (query.isError && items.length === 0) return <ErrorState onRetry={() => query.refetch()} />;
  if (items.length === 0 && !query.hasNextPage) return <EmptyState {...empty} />;

  return (
    <div>
      {items.map((item) => (
        // A tweet can appear twice: as posted and as someone's retweet.
        <Tweet key={`${item.retweet?.id ?? 'tweet'}-${item.tweet.id}`} tweet={item.tweet} retweet={item.retweet} />
      ))}
      <div ref={sentinel} />
      {query.isFetchingNextPage && <TweetSkeleton />}
      {query.isFetchNextPageError && <ErrorState onRetry={() => query.fetchNextPage()} />}
    </div>
  );
}
