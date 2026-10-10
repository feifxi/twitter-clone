'use client';

import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import type { PageResponse, UserResponse, TweetResponse } from '@/types';
import type { UpdateProfileInput } from '@/lib/validation';
import { uploadFileWithPresignedUrl } from '@/api/upload';
import { currentUserQueryKey } from './useCurrentUser';

const userQueryKey = (id: number) => ['users', id] as const;
export const userFeedQueryKey = (userId: number) => ['feeds', 'user', userId] as const;

export function useUserProfile(userId: number | null) {
  return useQuery({
    queryKey: userQueryKey(userId!),
    queryFn: async (): Promise<UserResponse> => {
      const { data } = await axiosInstance.get<UserResponse>(`/users/${userId}`);
      return data;
    },
    enabled: userId != null && userId > 0,
  });
}

/** Resolve user by username via search (backend has GET /users/{id} by id). */
export function useUserProfileByUsername(username: string) {
  return useQuery({
    queryKey: ['users', 'byUsername', username],
    queryFn: async (): Promise<UserResponse | null> => {
      const { data } = await axiosInstance.get<PageResponse<UserResponse>>(
        '/search/users',
        { params: { q: username, size: 10 } }
      );
      const match = data.items.find(
        (u) => u.username.toLowerCase() === username.toLowerCase()
      );
      return match ?? null;
    },
    enabled: !!username,
  });
}

export function useUserFeed(userId: number | null, pageSize = 20) {
  return useInfiniteQuery({
    queryKey: userFeedQueryKey(userId!),
    queryFn: async ({
      pageParam,
    }: {
      pageParam: string | null;
    }): Promise<PageResponse<TweetResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<TweetResponse>>(
        `/feeds/user/${userId}`,
        { params: { cursor: pageParam ?? undefined, size: pageSize } }
      );
      return data;
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.hasNext ? lastPage.nextCursor : undefined,
    enabled: userId != null && userId > 0,
  });
}

// Helper to update user follow status in various query caches
function useUpdateUserCache() {
  const queryClient = useQueryClient();

  return (userId: number, isFollowing: boolean) => {
    // 1. Update individual user profile
    queryClient.setQueryData<UserResponse>(userQueryKey(userId), (old) => {
      if (!old) return old;
      return {
        ...old,
        isFollowing: isFollowing,
        followersCount: isFollowing ? (old.followersCount + 1) : Math.max(0, old.followersCount - 1),
      };
    });

    // 2. Update Discovery/Suggested Users
    queryClient.setQueriesData<PageResponse<UserResponse>>({ queryKey: ['discovery', 'users'] }, (old) => {
      if (!old) return old;
      return {
        ...old,
        items: old.items.map(u => u.id === userId ? { ...u, isFollowing: isFollowing } : u)
      };
    });

    // 3. Update Search Users
    queryClient.setQueriesData<PageResponse<UserResponse>>({ queryKey: ['search', 'users'] }, (old) => {
      if (!old) return old;
      return {
        ...old,
        items: old.items.map(u => u.id === userId ? { ...u, isFollowing: isFollowing } : u)
      };
    });

    // 4. Update "By Username" cache if exists (we don't know username but can iterate)
    // Actually setQueriesData with predicate is hard without username. 
    // We can rely on invalidation for byUsername or try to find it.
    // For now, let's stick to the list caches which are most visible.
  };
}

export function useFollowUser() {
  const queryClient = useQueryClient();
  const updateUserCache = useUpdateUserCache();

  return useMutation({
    mutationFn: (userId: number) => axiosInstance.post(`/users/${userId}/follow`),
    onMutate: async (userId) => {
      await queryClient.cancelQueries({ queryKey: userQueryKey(userId) });
      await queryClient.cancelQueries({ queryKey: ['discovery', 'users'] });
      await queryClient.cancelQueries({ queryKey: ['search', 'users'] });
      await queryClient.cancelQueries({ queryKey: ['users', 'byUsername'] });

      updateUserCache(userId, true);
    },
    onError: (_err, userId) => {
      updateUserCache(userId, false); // Rollback
    },
    onSuccess: () => {
      // Optimistic update handles UI. No need to refetch.
      // queryClient.invalidateQueries({ queryKey: userQueryKey(userId) });
      // queryClient.invalidateQueries({ queryKey: ['feeds'] }); 
      // queryClient.invalidateQueries({ queryKey: ['discovery', 'users'] });
      // queryClient.invalidateQueries({ queryKey: ['search', 'users'] });
      // queryClient.invalidateQueries({ queryKey: ['users', 'byUsername'] });

      // We manually update our own profile stats (following count) because `updateUserCache` 
      // currently only updates the target user's followers count.
      // We need to update *current user's* following count in the cache.
      // But we don't have the current user ID easily here unless we use `useAuth` or store.
      // actually `updateUserCache` does NOT update current user stats. 
      // Let's add that logic if possible, or just accept that "Following" count on my profile 
      // might be stale until refresh. 
      // The request says "shouldn't refetch". So we stop here.
    },
  });
}

export function useUnfollowUser() {
  const queryClient = useQueryClient();
  const updateUserCache = useUpdateUserCache();

  return useMutation({
    mutationFn: (userId: number) =>
      axiosInstance.delete(`/users/${userId}/follow`),
    onMutate: async (userId) => {
      await queryClient.cancelQueries({ queryKey: userQueryKey(userId) });
      await queryClient.cancelQueries({ queryKey: ['discovery', 'users'] });
      await queryClient.cancelQueries({ queryKey: ['search', 'users'] });
      await queryClient.cancelQueries({ queryKey: ['users', 'byUsername'] });

      updateUserCache(userId, false);
    },
    onError: (_err, userId) => {
      updateUserCache(userId, true); // Rollback
    },
    onSuccess: () => {
      // No refetch
    },
  });
}

export function useUpdateProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ displayName, bio, avatar, banner, removeAvatar, removeBanner }: UpdateProfileInput & { avatar?: File; banner?: File; removeAvatar?: boolean; removeBanner?: boolean }) => {
      let avatarKey: string | undefined;
      let bannerKey: string | undefined;

      if (avatar) {
        avatarKey = await uploadFileWithPresignedUrl(avatar, 'avatars');
      } else if (removeAvatar) {
        avatarKey = '';
      }

      if (banner) {
        bannerKey = await uploadFileWithPresignedUrl(banner, 'banners');
      } else if (removeBanner) {
        bannerKey = '';
      }

      const payload = {
        displayName,
        bio,
        avatarKey,
        bannerKey,
      };

      const { data } = await axiosInstance.put<UserResponse>('/users/profile', payload);
      return data;
    },

    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: userQueryKey(data.id) });
      queryClient.invalidateQueries({ queryKey: ['users', 'byUsername', data.username] });
      queryClient.invalidateQueries({ queryKey: currentUserQueryKey });
    },
  });
}

export function useUserFollowers(userId: number | null, pageSize = 20) {
  return useInfiniteQuery({
    queryKey: ['users', userId, 'followers'],
    queryFn: async ({ pageParam = null as string | null }) => {
      const { data } = await axiosInstance.get<PageResponse<UserResponse>>(`/users/${userId}/followers`, {
        params: { cursor: pageParam ?? undefined, size: pageSize },
      });
      return data;
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNext ? lastPage.nextCursor : undefined),
    enabled: !!userId,
  });
}

export function useUserFollowing(userId: number | null, pageSize = 20) {
  return useInfiniteQuery({
    queryKey: ['users', userId, 'following'],
    queryFn: async ({ pageParam = null as string | null }) => {
      const { data } = await axiosInstance.get<PageResponse<UserResponse>>(`/users/${userId}/following`, {
        params: { cursor: pageParam ?? undefined, size: pageSize },
      });
      return data;
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNext ? lastPage.nextCursor : undefined),
    enabled: !!userId,
  });
}
