import { useMutation, useQueryClient } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import type { TweetResponse } from '@/types';
import { tweetQueryKey } from './useTweet';
import { useUpdateTweetCache } from './useOptimisticTweetUpdate';

import { useAuth } from '@/hooks/useAuth';

/**
 * Applies a (un)retweet to a cached tweet. Undoing removes the timeline entry
 * that exists only because the current user retweeted it.
 */
function toggleRetweetInTweet(t: TweetResponse, retweeted: boolean, currentUserId?: number): TweetResponse | null {
  if (!retweeted && currentUserId !== undefined && t.retweetedBy?.id === currentUserId) {
    return null;
  }
  return { ...t, isRetweeted: retweeted, retweetCount: Math.max(0, t.retweetCount + (retweeted ? 1 : -1)) };
}

export function useRetweet() {
  const queryClient = useQueryClient();
  const updateTweetCache = useUpdateTweetCache();
  const { user } = useAuth();
  const currentUserId = user?.id;

  return useMutation({
    mutationFn: async (tweetId: number) => {
      await axiosInstance.post(`/tweets/${tweetId}/retweet`);
    },
    onMutate: async (tweetId) => {
      await queryClient.cancelQueries({ queryKey: ['feeds'] });
      await queryClient.cancelQueries({ queryKey: ['tweets', tweetId] });
      await queryClient.cancelQueries({ queryKey: ['tweets'] });
      await queryClient.cancelQueries({ queryKey: ['search'] });

      updateTweetCache(tweetId, (t) => toggleRetweetInTweet(t, true, currentUserId));
    },
    onError: () => {
      queryClient.invalidateQueries({ queryKey: ['feeds'] });
      queryClient.invalidateQueries({ queryKey: ['tweets'] });
      queryClient.invalidateQueries({ queryKey: ['search'] });
    },
    onSettled: (_data, _err, tweetId) => {
      queryClient.invalidateQueries({ queryKey: tweetQueryKey(tweetId) });
    },
  });
}

export function useUnretweet() {
  const queryClient = useQueryClient();
  const updateTweetCache = useUpdateTweetCache();
  const { user } = useAuth();
  const currentUserId = user?.id;

  return useMutation({
    mutationFn: async (tweetId: number) => {
      await axiosInstance.delete(`/tweets/${tweetId}/retweet`);
    },
    onMutate: async (tweetId) => {
      await queryClient.cancelQueries({ queryKey: ['feeds'] });
      await queryClient.cancelQueries({ queryKey: ['tweets', tweetId] });
      await queryClient.cancelQueries({ queryKey: ['tweets'] });
      await queryClient.cancelQueries({ queryKey: ['search'] });

      updateTweetCache(tweetId, (t) => toggleRetweetInTweet(t, false, currentUserId));
    },
    onError: () => {
      queryClient.invalidateQueries({ queryKey: ['feeds'] });
      queryClient.invalidateQueries({ queryKey: ['tweets'] });
      queryClient.invalidateQueries({ queryKey: ['search'] });
    },
    onSettled: (_data, _err, tweetId) => {
      queryClient.invalidateQueries({ queryKey: tweetQueryKey(tweetId) });
    },
  });
}
