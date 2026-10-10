'use client';

import { useCallback } from 'react';
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { axiosInstance } from '@/api/axiosInstance';
import { uploadFileWithPresignedUrl } from '@/api/upload';
import { useAuth } from '@/hooks/useAuth';
import { debouncedToggle } from '@/lib/debounced-toggle';
import { invalidateEntities, updateUser } from '@/lib/query-cache';
import { PAGE_SIZE, queryKeys, type ProfileTab } from '@/lib/query-keys';
import type { UpdateProfileInput } from '@/lib/validation';
import type { PageResponse, TweetResponse, UserResponse } from '@/types';

export function useUserProfile(userId: number | null) {
  return useQuery({
    queryKey: queryKeys.users.detail(userId!),
    queryFn: async (): Promise<UserResponse> => {
      const { data } = await axiosInstance.get<UserResponse>(`/users/${userId}`);
      return data;
    },
    enabled: userId != null && userId > 0,
  });
}

export function useUserProfileByUsername(username: string) {
  return useQuery({
    queryKey: queryKeys.users.byUsername(username),
    queryFn: async (): Promise<UserResponse> => {
      const { data } = await axiosInstance.get<UserResponse>(`/users/by-username/${encodeURIComponent(username)}`);
      return data;
    },
    enabled: !!username,
  });
}

/** A profile tab. Likes are private: only the owner can load their own. */
export function useUserTimeline(userId: number | null, tab: ProfileTab = 'posts', pageSize = PAGE_SIZE) {
  return useInfiniteQuery({
    queryKey: queryKeys.users.timeline(userId!, tab),
    queryFn: async ({ pageParam }: { pageParam: string | null }): Promise<PageResponse<TweetResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<TweetResponse>>(`/users/${userId}/${tab}`, {
        params: { cursor: pageParam ?? undefined, size: pageSize },
      });
      return data;
    },
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNext ? lastPage.nextCursor : undefined),
    enabled: userId != null && userId > 0,
  });
}

/**
 * Follow/unfollow. Every cached copy of the user (profile, lists, tweet authors)
 * and the current user's following count update immediately; the request is debounced.
 */
export function useToggleFollow() {
  const queryClient = useQueryClient();
  const { user: me } = useAuth();
  const myId = me?.id;

  return useCallback(
    (target: Pick<UserResponse, 'id' | 'isFollowing'>) => {
      const following = !target.isFollowing;
      const delta = following ? 1 : -1;
      updateUser(queryClient, target.id, (u) => ({
        ...u,
        isFollowing: following,
        followersCount: Math.max(0, u.followersCount + delta),
      }));
      if (myId !== undefined) {
        updateUser(queryClient, myId, (u) => ({ ...u, followingCount: Math.max(0, u.followingCount + delta) }));
      }

      debouncedToggle(`follow:${target.id}`, target.isFollowing, following, async (state) => {
        try {
          if (state) await axiosInstance.post(`/users/${target.id}/follow`);
          else await axiosInstance.delete(`/users/${target.id}/follow`);
          // The following timeline gains or loses this user's tweets.
          queryClient.invalidateQueries({ queryKey: queryKeys.feeds.following, refetchType: 'none' });
        } catch {
          toast.error(state ? 'Could not follow' : 'Could not unfollow');
          invalidateEntities(queryClient);
        }
      });
    },
    [queryClient, myId],
  );
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
      // Name, avatar and bio show up on every tweet of theirs: refresh all copies.
      updateUser(queryClient, data.id, (u) => ({ ...u, ...data }));
      queryClient.invalidateQueries({ queryKey: queryKeys.auth.me });
    },
  });
}

export function useUserFollowers(userId: number | null, pageSize = 20) {
  return useInfiniteQuery({
    queryKey: queryKeys.users.followers(userId!),
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
    queryKey: queryKeys.users.following(userId!),
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
