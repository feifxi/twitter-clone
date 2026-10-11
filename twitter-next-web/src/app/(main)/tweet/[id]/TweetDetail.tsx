'use client';

import { PageHeader } from '@/components/PageHeader';
import { Tweet, TweetSkeleton } from '@/components/Tweet';
import { CreateTweet } from '@/components/CreateTweet';
import { useTweet, useRepliesInfinite } from '@/hooks/useTweet';

/** The tweet and its first replies arrive prefetched from the server page. */
export function TweetDetail({ tweetId }: { tweetId: number }) {
  const { data: tweet, isLoading: tweetLoading } = useTweet(tweetId);
  const repliesQuery = useRepliesInfinite(tweetId);
  const replyList = repliesQuery.data?.pages.flatMap((p) => p.items) ?? [];

  return (
    <div className="min-h-screen">
      <PageHeader title="Post" back />

      {tweetLoading || !tweet ? (
        <TweetSkeleton />
      ) : (
        <>
          <Tweet tweet={tweet} variant="detail" />
          <div className="border-b border-border">
            <CreateTweet isReply replyToId={tweet.id} placeholder="Post your reply" className="border-none px-4 py-2" />
          </div>
          {repliesQuery.isLoading ? (
            <>
              <TweetSkeleton />
              <TweetSkeleton />
            </>
          ) : (
            replyList.map((reply) => <Tweet key={reply.id} tweet={reply} />)
          )}
        </>
      )}
    </div>
  );
}
