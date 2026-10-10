'use client';

import { useInfiniteQuery, useMutation, useQuery, useQueryClient, type InfiniteData } from '@tanstack/react-query';
import { useAuth } from '@/hooks/useAuth';
import { axiosInstance } from '@/api/axiosInstance';
import { queryKeys } from '@/lib/query-keys';
import type { PageResponse, NotificationResponse } from '@/types';

export function useNotifications(pageSize = 20) {
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
        staleTime: 300000, // 5 minutes
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

export function useMarkAllRead() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: () => axiosInstance.post('/notifications/mark-read'),
        onSuccess: () => {
            queryClient.setQueryData(queryKeys.notifications.unreadCount, 0);
            queryClient.setQueryData<InfiniteData<PageResponse<NotificationResponse>>>(
                queryKeys.notifications.list,
                (old) =>
                    old && {
                        ...old,
                        pages: old.pages.map((page) => ({
                            ...page,
                            items: page.items.map((n) => ({ ...n, isRead: true })),
                        })),
                    },
            );
        },
    });
}
