'use client';

import { useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { axiosInstance, isUnauthorized } from '@/api/axiosInstance';
import { queryKeys } from '@/lib/query-keys';
import type { CurrentUserResponse } from '@/types';

async function fetchCurrentUser(): Promise<CurrentUserResponse | null> {
  try {
    const { data } = await axiosInstance.get<CurrentUserResponse>('/auth/me');
    return data;
  } catch (error) {
    if (isUnauthorized(error)) return null;
    throw error;
  }
}

/**
 * The signed-in user (null when signed out). The server render seeds it, so it
 * is known on the first paint; the session itself lives in httpOnly cookies.
 */
export function useAuth() {
  const queryClient = useQueryClient();
  const router = useRouter();

  const { data: user = null, isPending } = useQuery({
    queryKey: queryKeys.auth.me,
    queryFn: fetchCurrentUser,
    staleTime: 5 * 60_000,
  });

  const logout = useCallback(async () => {
    try {
      // Ends only this browser's session and clears its cookies.
      await axiosInstance.post('/auth/logout');
    } finally {
      queryClient.clear();
      queryClient.setQueryData(queryKeys.auth.me, null);
      sessionStorage.clear();
      router.refresh(); // re-render server components as signed out
    }
  }, [queryClient, router]);

  return { user, isLoggedIn: user !== null, isLoading: isPending, logout };
}
