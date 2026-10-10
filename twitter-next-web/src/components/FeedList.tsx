'use client';

import { useRef, useEffect } from 'react';
import { Tweet, TweetSkeleton } from './Tweet';
import type { TweetResponse } from '@/types';

interface FeedListProps {
  tweets: TweetResponse[];
  isLoading: boolean;
  isFetchingNextPage?: boolean;
  hasNextPage?: boolean;
  fetchNextPage?: () => void;
}

export function FeedList({
  tweets,
  isLoading,
  isFetchingNextPage,
  hasNextPage,
  fetchNextPage,
}: FeedListProps) {
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hasNextPage || !fetchNextPage || !sentinelRef.current) return;
    const el = sentinelRef.current;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) fetchNextPage();
      },
      { rootMargin: '200px' }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasNextPage, fetchNextPage]);

  if (isLoading) {
    return (
      <div className="flex flex-col">
        {Array.from({ length: 5 }).map((_, i) => (
          <TweetSkeleton key={i} />
        ))}
      </div>
    );
  }

  if (tweets.length === 0) {
    return (
      <div className="py-12 text-center text-muted-foreground text-[15px]">
        No tweets yet.
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      {tweets.map((tweet) => (
        // A tweet can appear twice: as posted and as someone's retweet.
        <Tweet key={`${tweet.retweetedBy?.id ?? 0}-${tweet.id}`} tweet={tweet} />
      ))}
      <div ref={sentinelRef} />
      {isFetchingNextPage && (
        <div className="flex justify-center py-2">
          <TweetSkeleton />
        </div>
      )}
      {hasNextPage && !isFetchingNextPage && fetchNextPage && (
        <div className="flex justify-center py-4 border-b border-border">
          <button
            type="button"
            onClick={() => fetchNextPage()}
            className="py-2 px-4 rounded-full text-[15px] font-medium text-primary hover:bg-primary/10 transition-colors"
          >
            Load more
          </button>
        </div>
      )}
    </div>
  );
}
