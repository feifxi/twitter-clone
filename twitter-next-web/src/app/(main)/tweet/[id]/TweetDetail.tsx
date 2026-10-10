'use client';

import { useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { Tweet, TweetSkeleton } from '@/components/Tweet';
import { CreateTweet } from '@/components/CreateTweet';
import { useTweet, useRepliesInfinite } from '@/hooks/useTweet';

/** The tweet and its first replies arrive prefetched from the server page. */
export function TweetDetail({ tweetId }: { tweetId: number }) {
  const router = useRouter();
  const { data: tweet, isLoading: tweetLoading } = useTweet(tweetId);
  const repliesQuery = useRepliesInfinite(tweetId);
  const replyList = repliesQuery.data?.pages.flatMap((p) => p.items) ?? [];

  return (
    <div className="min-h-screen">
      {/* Sticky Header */}
      <div className="sticky top-0 z-30 bg-background/60 backdrop-blur-md border-b border-border px-4 py-3 flex items-center gap-6">
        <button
          onClick={() => router.back()}
          className="p-2 -ml-2 rounded-full hover:bg-card transition-colors"
          aria-label="Back"
        >
          <ArrowLeft className="w-5 h-5 text-foreground" />
        </button>
        <h1 className="text-[20px] font-bold text-foreground">Post</h1>
      </div>

      {tweetLoading || !tweet ? (
        <div className="border-b border-border">
          <TweetSkeleton />
        </div>
      ) : (
        <>
          <Tweet tweet={tweet} disableClick />
          <div className="border-b border-border">
            <CreateTweet
              isReply
              replyToId={tweet.id}
              placeholder="Post your reply"
              className="px-4 py-2 border-none"
            />
          </div>
          <p className="px-4 py-2 text-muted-foreground text-[15px] font-medium">
            Replies
          </p>
          {repliesQuery.isLoading ? (
            <div>
              <TweetSkeleton />
              <TweetSkeleton />
            </div>
          ) : replyList.length === 0 ? (
            <p className="p-4 text-muted-foreground text-[15px]">No replies yet.</p>
          ) : (
            <div>
              {replyList.map((r) => (
                <Tweet key={r.id} tweet={r} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}
