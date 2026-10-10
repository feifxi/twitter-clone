import { useQuery } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import type { CurrentUserResponse } from '@/types';

export const currentUserQueryKey = ['auth', 'me'] as const;

export function useCurrentUser() {
  return useQuery({
    queryKey: currentUserQueryKey,
    queryFn: async (): Promise<CurrentUserResponse> => {
      const { data } = await axiosInstance.get<CurrentUserResponse>('/auth/me');
      return data;
    },
    staleTime: 5 * 60 * 1000,
  });
}
