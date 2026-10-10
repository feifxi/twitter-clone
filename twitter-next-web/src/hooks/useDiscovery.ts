'use client';

import { useQuery } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import { queryKeys } from '@/lib/query-keys';
import type { PageResponse, HashtagResponse, UserResponse } from '@/types';

export function useTrendingHashtags(limit = 10) {
  return useQuery({
    queryKey: queryKeys.discovery.trending(limit),
    queryFn: async (): Promise<HashtagResponse[]> => {
      const { data } = await axiosInstance.get<HashtagResponse[]>(
        '/discovery/trending',
        { params: { limit } }
      );
      return data;
    },
    staleTime: 2 * 60 * 1000,
  });
}

export function useSuggestedUsers(pageSize = 5) {
  return useQuery({
    queryKey: queryKeys.discovery.users(pageSize),
    queryFn: async (): Promise<PageResponse<UserResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<UserResponse>>(
        '/discovery/users',
        { params: { size: pageSize } }
      );
      return data;
    },
    staleTime: 2 * 60 * 1000,
  });
}

export function useSearchUsers(query: string, enabled = true, size = 5) {
  return useQuery({
    queryKey: queryKeys.search.users(query, size),
    queryFn: async (): Promise<PageResponse<UserResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<UserResponse>>(
        '/search/users',
        { params: { q: query, size } }
      );
      return data;
    },
    enabled: query.length > 0 && enabled,
    staleTime: 60 * 1000,
  });
}

export function useSearchHashtags(query: string) {
  return useQuery({
    queryKey: queryKeys.search.hashtags(query),
    queryFn: async (): Promise<HashtagResponse[]> => {
      const { data } = await axiosInstance.get<HashtagResponse[]>(
        '/search/hashtags',
        { params: { q: query, limit: 10 } }
      );
      return data;
    },
    enabled: query.length > 0,
    staleTime: 60 * 1000,
  });
}
