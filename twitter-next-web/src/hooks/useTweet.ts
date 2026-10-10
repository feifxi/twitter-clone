'use client';

import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from '@tanstack/react-query';
import { toast } from 'sonner';
import { axiosInstance } from '@/api/axiosInstance';
import { uploadFileWithPresignedUrl } from '@/api/upload';
import { useAuth } from '@/hooks/useAuth';
import { snapshotEntities, updateTweet } from '@/lib/query-cache';
import { queryKeys } from '@/lib/query-keys';
import type { TweetRequestInput } from '@/lib/validation';
import type { PageResponse, TweetResponse } from '@/types';

export function useTweet(tweetId: number | null) {
  return useQuery({
    queryKey: queryKeys.tweets.detail(tweetId!),
    queryFn: async (): Promise<TweetResponse> => {
      const { data } = await axiosInstance.get<TweetResponse>(`/tweets/${tweetId}`);
      return data;
    },
    enabled: tweetId != null && tweetId > 0,
  });
}

export function useRepliesInfinite(tweetId: number | null, pageSize = 20) {
  return useInfiniteQuery({
    queryKey: queryKeys.tweets.replies(tweetId!),
    queryFn: async ({ pageParam }: { pageParam: string | null }): Promise<PageResponse<TweetResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<TweetResponse>>(`/tweets/${tweetId}/replies`, {
        params: { cursor: pageParam ?? undefined, size: pageSize },
      });
      return data;
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNext ? lastPage.nextCursor : undefined),
    enabled: tweetId != null && tweetId > 0,
  });
}

export function useCreateTweet() {
  const queryClient = useQueryClient();
  const { user } = useAuth();

  return useMutation({
    mutationFn: async ({ content, media, parentId }: TweetRequestInput & { media?: File }) => {
      const mediaKey = media ? await uploadFileWithPresignedUrl(media, 'tweets') : undefined;
      const { data } = await axiosInstance.post<TweetResponse>('/tweets', { content, parentId, mediaKey });
      return data;
    },
    onSuccess: (tweet, { parentId }) => {
      // Show the new tweet at the top of the lists it belongs to, without refetching (and losing the scroll position).
      const prepend = (queryKey: readonly unknown[]) =>
        queryClient.setQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKey, (old) => {
          if (!old?.pages.length) return old;
          const [first, ...rest] = old.pages;
          return { ...old, pages: [{ ...first, items: [tweet, ...first.items] }, ...rest] };
        });

      if (parentId) {
        prepend(queryKeys.tweets.replies(parentId));
        updateTweet(queryClient, parentId, (t) => ({ ...t, replyCount: t.replyCount + 1 }));
        if (user) prepend(queryKeys.users.timeline(user.id, 'replies'));
        return;
      }
      prepend(queryKeys.feeds.global);
      prepend(queryKeys.feeds.following);
      if (user) prepend(queryKeys.users.timeline(user.id, 'posts'));
      if (user && tweet.mediaUrl) prepend(queryKeys.users.timeline(user.id, 'media'));
    },
  });
}

export function useDeleteTweet() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (tweet: TweetResponse) => {
      await axiosInstance.delete(`/tweets/${tweet.id}`);
    },
    onMutate: async (tweet) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.feeds.all });
      await queryClient.cancelQueries({ queryKey: queryKeys.tweets.all });
      const rollback = snapshotEntities(queryClient);

      updateTweet(queryClient, tweet.id, () => null);
      if (tweet.replyToTweetId) {
        updateTweet(queryClient, tweet.replyToTweetId, (t) => ({ ...t, replyCount: Math.max(0, t.replyCount - 1) }));
      }
      return { rollback };
    },
    onError: (_err, _tweet, context) => {
      context?.rollback();
      toast.error('Failed to delete tweet');
    },
    onSuccess: (_data, tweet) => {
      toast.success('Tweet deleted');
      queryClient.removeQueries({ queryKey: queryKeys.tweets.detail(tweet.id) });
    },
  });
}
