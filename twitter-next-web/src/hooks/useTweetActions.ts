'use client';

import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { axiosInstance } from '@/api/axiosInstance';
import { useAuth } from '@/hooks/useAuth';
import { debouncedToggle } from '@/lib/debounced-toggle';
import { invalidateEntities, updateTweet } from '@/lib/query-cache';
import { queryKeys } from '@/lib/query-keys';
import type { TweetResponse } from '@/types';

/**
 * Like/unlike. The cache (every copy of the tweet) flips immediately; the
 * request is debounced, so render `tweet.isLiked` / `tweet.likeCount` as is.
 */
export function useToggleLike() {
  const queryClient = useQueryClient();

  return useCallback(
    (tweet: TweetResponse) => {
      const liked = !tweet.isLiked;
      updateTweet(queryClient, tweet.id, (t) => ({
        ...t,
        isLiked: liked,
        likeCount: Math.max(0, t.likeCount + (liked ? 1 : -1)),
      }));

      debouncedToggle(`like:${tweet.id}`, tweet.isLiked, liked, async (state) => {
        try {
          if (state) await axiosInstance.post(`/tweets/${tweet.id}/like`);
          else await axiosInstance.delete(`/tweets/${tweet.id}/like`);
        } catch {
          toast.error(state ? 'Could not like the tweet' : 'Could not remove the like');
          invalidateEntities(queryClient); // restore the server's truth
        }
      });
    },
    [queryClient],
  );
}

/**
 * Retweet/undo. Undoing also removes the timeline entries that exist only
 * because the current user retweeted.
 */
export function useToggleRetweet() {
  const queryClient = useQueryClient();
  const { user } = useAuth();
  const myId = user?.id;

  return useCallback(
    (tweet: TweetResponse) => {
      const retweeted = !tweet.isRetweeted;
      updateTweet(queryClient, tweet.id, (t) => {
        if (!retweeted && t.retweetedBy?.id === myId) return null;
        return { ...t, isRetweeted: retweeted, retweetCount: Math.max(0, t.retweetCount + (retweeted ? 1 : -1)) };
      });

      debouncedToggle(`retweet:${tweet.id}`, tweet.isRetweeted, retweeted, async (state) => {
        try {
          if (state) await axiosInstance.post(`/tweets/${tweet.id}/retweet`);
          else await axiosInstance.delete(`/tweets/${tweet.id}/retweet`);
          // A new retweet appears in timelines (mine, my followers'): fetch them fresh next time.
          queryClient.invalidateQueries({ queryKey: queryKeys.feeds.following, refetchType: 'none' });
        } catch {
          toast.error(state ? 'Could not retweet' : 'Could not undo the retweet');
          invalidateEntities(queryClient);
        }
      });
    },
    [queryClient, myId],
  );
}
