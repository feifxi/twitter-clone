'use client';

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuthStore } from '@/store/useAuthStore';
import { axiosInstance } from '@/api/axiosInstance';
import type { CurrentUserResponse } from '@/types';
import { currentUserQueryKey } from './useCurrentUser';

/** Sync store with server on mount when we have a token (e.g. after refresh). */
export function useAuth() {
  const { user, accessToken, setAuth, logout } = useAuthStore();
  const queryClient = useQueryClient();

  const { data: serverUser, isLoading } = useQuery({
    queryKey: [...currentUserQueryKey, accessToken ?? 'none'],
    queryFn: async (): Promise<CurrentUserResponse> => {
      const { data } = await axiosInstance.get<CurrentUserResponse>('/auth/me');
      return data;
    },
    enabled: !!accessToken,
    staleTime: 5 * 60 * 1000,
    retry: false,
  });

  const resolvedUser = serverUser ?? user;
  const isLoggedIn = !!accessToken && !!resolvedUser;

  async function doLogout() {
    try {
      // Ends only this device's session.
      await axiosInstance.post('/auth/logout', { refreshToken: useAuthStore.getState().refreshToken });
    } finally {
      logout();
      queryClient.clear();
      sessionStorage.clear();
    }
  }

  return {
    user: resolvedUser,
    accessToken,
    setAuth,
    logout: doLogout,
    isLoading: !!accessToken && isLoading,
    isLoggedIn,
  };
}
