'use client';

import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/hooks/useAuth';
import { axiosInstance } from '@/api/axiosInstance';
import { queryKeys } from '@/lib/query-keys';
import type { PageResponse, NotificationResponse } from '@/types';

export function useNotifications(pageSize = 20) {
    const { isLoggedIn } = useAuth();
    return useInfiniteQuery({
        queryKey: queryKeys.notifications.list,
        queryFn: async ({ pageParam }: { pageParam: string | null }): Promise<PageResponse<NotificationResponse>> => {
            const { data } = await axiosInstance.get<PageResponse<NotificationResponse>>(
                '/notifications',
                { params: { cursor: pageParam ?? undefined, size: pageSize } }
            );
            return data;
        },
        initialPageParam: null as string | null,
        getNextPageParam: (lastPage) =>
            lastPage.hasNext ? lastPage.nextCursor : undefined,
        staleTime: 300000, // 5 minutes; SSE events invalidate it sooner
        enabled: isLoggedIn,
    });
}

export function useUnreadCount() {
    const { isLoggedIn } = useAuth();
    return useQuery({
        queryKey: queryKeys.notifications.unreadCount,
        queryFn: async (): Promise<number> => {
            const { data } = await axiosInstance.get<{ count: number }>('/notifications/unread-count');
            return data.count;
        },
        refetchInterval: 300000, // Poll every 5 minutes as fallback, rely on SSE
        staleTime: 30000,
        refetchOnWindowFocus: false,
        refetchOnReconnect: false,
        enabled: isLoggedIn,
    });
}

/**
 * Marks everything read on the server and clears the badge. Like X, the list on
 * screen keeps its unread highlights for this visit; the next visit shows them read.
 */
export function useMarkAllRead() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: () => axiosInstance.post('/notifications/mark-read'),
        onSuccess: () => {
            queryClient.setQueryData(queryKeys.notifications.unreadCount, 0);
            queryClient.invalidateQueries({ queryKey: queryKeys.notifications.list, refetchType: 'none' });
        },
    });
}
