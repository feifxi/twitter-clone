import { useMutation, useQueryClient } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import type { TweetResponse } from '@/types';
import { tweetQueryKey } from './useTweet';
import { useUpdateTweetCache } from './useOptimisticTweetUpdate';

function toggleLikeInTweet(t: TweetResponse, liked: boolean): TweetResponse {
  return { ...t, isLiked: liked, likeCount: Math.max(0, t.likeCount + (liked ? 1 : -1)) };
}

export function useLikeTweet() {
  const queryClient = useQueryClient();
  const updateTweetCache = useUpdateTweetCache();

  return useMutation({
    mutationFn: async (tweetId: number) => {
      await axiosInstance.post(`/tweets/${tweetId}/like`);
    },
    onMutate: async (tweetId) => {
      await queryClient.cancelQueries({ queryKey: ['feeds'] });
      await queryClient.cancelQueries({ queryKey: ['tweets', tweetId] });
      await queryClient.cancelQueries({ queryKey: ['tweets'] }); // Cancel replies too if needed
      await queryClient.cancelQueries({ queryKey: ['search'] });

      updateTweetCache(tweetId, (t) => toggleLikeInTweet(t, true));
    },
    onError: () => {
      // Rollback is complex with global helper. 
      // Simplest strategy: Invalidate everything on error to restore truth.
      queryClient.invalidateQueries({ queryKey: ['feeds'] });
      queryClient.invalidateQueries({ queryKey: ['tweets'] });
      queryClient.invalidateQueries({ queryKey: ['search'] });
    },
    onSettled: (_data, _err, tweetId) => {
      // Only invalidate specific tweet to respect scroll position
      queryClient.invalidateQueries({ queryKey: tweetQueryKey(tweetId) });
      // Do NOT invalidate feeds or search to avoid scroll jump
    },
  });
}

export function useUnlikeTweet() {
  const queryClient = useQueryClient();
  const updateTweetCache = useUpdateTweetCache();

  return useMutation({
    mutationFn: async (tweetId: number) => {
      await axiosInstance.delete(`/tweets/${tweetId}/like`);
    },
    onMutate: async (tweetId) => {
      await queryClient.cancelQueries({ queryKey: ['feeds'] });
      await queryClient.cancelQueries({ queryKey: ['tweets', tweetId] });
      await queryClient.cancelQueries({ queryKey: ['tweets'] });
      await queryClient.cancelQueries({ queryKey: ['search'] });

      updateTweetCache(tweetId, (t) => toggleLikeInTweet(t, false));
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
