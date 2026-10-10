'use client';

import type { InfiniteData, UseInfiniteQueryResult } from '@tanstack/react-query';
import { EmptyState, ErrorState, type EmptyCopy } from '@/components/ListStates';
import { Tweet, TweetSkeleton } from '@/components/Tweet';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import type { PageResponse, TweetResponse } from '@/types';

interface FeedListProps {
  query: UseInfiniteQueryResult<InfiniteData<PageResponse<TweetResponse>>>;
  empty: EmptyCopy;
  /** Optional client-side filter over loaded tweets. */
  filter?: (tweet: TweetResponse) => boolean;
}

/** An infinitely scrolling list of tweets with loading, error and empty states. */
export function FeedList({ query, empty, filter }: FeedListProps) {
  const sentinel = useInfiniteScroll(query);
  const all = query.data?.pages.flatMap((page) => page.items) ?? [];
  const tweets = filter ? all.filter(filter) : all;

  if (query.isPending) {
    return Array.from({ length: 5 }, (_, i) => <TweetSkeleton key={i} />);
  }
  if (query.isError && tweets.length === 0) return <ErrorState onRetry={() => query.refetch()} />;
  if (tweets.length === 0 && !query.hasNextPage) return <EmptyState {...empty} />;

  return (
    <div>
      {tweets.map((tweet) => (
        // A tweet can appear twice: as posted and as someone's retweet.
        <Tweet key={`${tweet.retweetedBy?.id ?? 0}-${tweet.id}`} tweet={tweet} />
      ))}
      <div ref={sentinel} />
      {query.isFetchingNextPage && <TweetSkeleton />}
      {query.isFetchNextPageError && <ErrorState onRetry={() => query.fetchNextPage()} />}
    </div>
  );
}
