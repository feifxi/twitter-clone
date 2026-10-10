'use client';

import {
  useQuery,
  useInfiniteQuery,
  useQueryClient,
  useMutation,
  type QueryKey,
} from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import type { PageResponse, TweetResponse } from '@/types';
import type { TweetRequestInput } from '@/lib/validation';
import { uploadFileWithPresignedUrl } from '@/api/upload';
import { toast } from 'sonner';

export const tweetQueryKey = (id: number) => ['tweets', id] as const;

export function useTweet(tweetId: number | null) {
  return useQuery({
    queryKey: tweetQueryKey(tweetId!),
    queryFn: async (): Promise<TweetResponse> => {
      const { data } = await axiosInstance.get<TweetResponse>(`/tweets/${tweetId}`);
      return data;
    },
    enabled: tweetId != null && tweetId > 0,
  });
}

export function useRepliesInfinite(tweetId: number | null, pageSize = 20) {
  return useInfiniteQuery({
    queryKey: ['tweets', tweetId, 'replies'],
    queryFn: async ({
      pageParam,
    }: {
      pageParam: string | null;
    }): Promise<PageResponse<TweetResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<TweetResponse>>(
        `/tweets/${tweetId}/replies`,
        { params: { cursor: pageParam ?? undefined, size: pageSize } }
      );
      return data;
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.hasNext ? lastPage.nextCursor : undefined,
    enabled: tweetId != null && tweetId > 0,
  });
}

export function useInvalidateTweet() {
  const queryClient = useQueryClient();
  return (tweetId: number) =>
    queryClient.invalidateQueries({ queryKey: tweetQueryKey(tweetId) });
}

import { useAuthStore } from '@/store/useAuthStore';
import { feedQueryKey } from './useFeed';
import { userFeedQueryKey } from './useProfile';
import type { InfiniteData } from '@tanstack/react-query';

export function useCreateTweet() {
  const queryClient = useQueryClient();
  const { user } = useAuthStore();

  return useMutation({
    mutationFn: async ({ content, media, parentId }: TweetRequestInput & { media?: File }) => {
      const mediaKey = media ? await uploadFileWithPresignedUrl(media, 'tweets') : undefined;
      const payload = { content, parentId, mediaKey };

      const { data } = await axiosInstance.post<TweetResponse>('/tweets', payload);
      return data;
    },
    onMutate: async () => {
      // Don't cancel queries, we're not doing optimistic updates anymore
    },
    onError: (_err, _newTweet) => {
      // Invalidate replies query
      if (_newTweet.parentId) {
        queryClient.invalidateQueries({ queryKey: ['tweets', _newTweet.parentId, 'replies'] });
      }
    },
    onSuccess: (realTweet, variables) => {
      // Helper to instantly inject the real tweet at the top of a feed
      const prependToFeed = (key: QueryKey) => {
        queryClient.setQueryData<InfiniteData<PageResponse<TweetResponse>>>(key, (old) => {
          if (!old) return old;

          const newPages = old.pages.map(page => ({ ...page }));

          if (newPages.length > 0) {
            newPages[0] = {
              ...newPages[0],
              items: [realTweet, ...newPages[0].items]
            };
          }

          return { ...old, pages: newPages };
        });
      };

      if (!variables.parentId) {
        prependToFeed(feedQueryKey('global'));
        prependToFeed(feedQueryKey('following'));
        // Update user's profile feed if viewing their own profile
        if (user) {
          prependToFeed(userFeedQueryKey(user.id));
        }
      } else {
        // Also update replies if it was a reply
        prependToFeed(['tweets', variables.parentId, 'replies']);
        // Ensure parent tweet reply count is visually incremented
        queryClient.setQueryData<TweetResponse>(tweetQueryKey(variables.parentId), (old) => {
          if (!old) return old;
          return { ...old, replyCount: old.replyCount + 1 };
        });
      }
    }
  });
}

export function useDeleteTweet() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (tweetId: number) => {
      await axiosInstance.delete(`/tweets/${tweetId}`);
    },
    onMutate: async (tweetId) => {
      // 1. Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: ['feeds'] });
      await queryClient.cancelQueries({ queryKey: ['tweets'] });

      // 2. Snapshot current data for rollback
      const snapshot: [QueryKey, InfiniteData<PageResponse<TweetResponse>> | TweetResponse][] = [];
      
      // Helper to capture and update infinite data
      const updateInfiniteFeeds = () => {
        const queries = queryClient.getQueriesData<InfiniteData<PageResponse<TweetResponse>>>({ queryKey: ['feeds'] });
        queries.forEach(([queryKey, oldData]) => {
          if (oldData) {
            snapshot.push([queryKey, oldData]);
            queryClient.setQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKey, (old) => {
                if (!old) return old;
                return {
                  ...old,
                  pages: old.pages.map((page) => ({
                    ...page,
                    items: page.items.filter((t) => t.id !== tweetId),
                  })),
                };
            });
          }
        });
      };

      // Helper to capture and update replies lists
      const updateRepliesLists = () => {
        // Find all reply queries: ['tweets', number, 'replies']
        const queries = queryClient.getQueriesData<InfiniteData<PageResponse<TweetResponse>>>({ 
            predicate: (query) => query.queryKey[0] === 'tweets' && query.queryKey[2] === 'replies'
        });
        
        queries.forEach(([queryKey, oldData]) => {
            if (oldData) {
              snapshot.push([queryKey, oldData]);
              queryClient.setQueryData<InfiniteData<PageResponse<TweetResponse>>>(queryKey, (old) => {
                  if (!old) return old;
                  return {
                    ...old,
                    pages: old.pages.map((page) => ({
                      ...page,
                      items: page.items.filter((t) => t.id !== tweetId),
                    })),
                  };
              });
            }
        });
      };

      // 3. Perform manual updates
      updateInfiniteFeeds();
      updateRepliesLists();

      // 4. Handle parent reply counter decrement
      // We need to find the tweet in cache to see if it has a parentId
      const allTweets = queryClient.getQueriesData<TweetResponse>({ queryKey: ['tweets'] });
      let parentId: number | null = null;
      
      for (const [, tweet] of allTweets) {
          if (tweet?.id === tweetId) {
              parentId = tweet.replyToTweetId ?? null;
              break;
          }
      }

      if (parentId) {
          const parentKey = tweetQueryKey(parentId);
          const oldParent = queryClient.getQueryData<TweetResponse>(parentKey);
          if (oldParent) {
              snapshot.push([parentKey, oldParent]);
              queryClient.setQueryData<TweetResponse>(parentKey, {
                  ...oldParent,
                  replyCount: Math.max(0, oldParent.replyCount - 1)
              });
          }
      }

      return { snapshot };
    },
    onError: (_err, _tweetId, context) => {
      // Rollback
      if (context?.snapshot) {
          context.snapshot.forEach(([key, val]) => {
              queryClient.setQueryData(key, val);
          });
      }
      toast.error('Failed to delete tweet');
    },
    onSuccess: () => {
      toast.success('Tweet deleted');
      // No global refetch needed! Optimistic update handles it.
    },
  });
}
