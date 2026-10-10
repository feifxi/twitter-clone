import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { axiosInstance } from '@/api/axiosInstance';
import { useAuth } from '@/hooks/useAuth';
import { queryKeys } from '@/lib/query-keys';
import type { ConversationResponse, MessageResponse, PageResponse } from '@/types';

export function useConversations(pageSize = 20) {
  const { isLoggedIn } = useAuth();
  return useInfiniteQuery({
    queryKey: queryKeys.messages.conversations,
    queryFn: async ({ pageParam }: { pageParam: string | null }): Promise<PageResponse<ConversationResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<ConversationResponse>>('/messages/conversations', {
        params: { cursor: pageParam ?? undefined, size: pageSize },
      });
      return data;
    },
    enabled: isLoggedIn,
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNext ? lastPage.nextCursor : undefined),
    staleTime: 30000,
  });
}

export function useConversationMessages(conversationId: number | null, pageSize = 30) {
  const { isLoggedIn } = useAuth();
  return useInfiniteQuery({
    queryKey: queryKeys.messages.conversation(conversationId ?? 0),
    queryFn: async ({ pageParam }: { pageParam: string | null }): Promise<PageResponse<MessageResponse>> => {
      const { data } = await axiosInstance.get<PageResponse<MessageResponse>>(
        `/messages/conversations/${conversationId}/messages`,
        { params: { cursor: pageParam ?? undefined, size: pageSize } },
      );
      return data;
    },
    enabled: !!conversationId && isLoggedIn,
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => (lastPage.hasNext ? lastPage.nextCursor : undefined),
    staleTime: 10000,
  });
}

export function useSendMessageToConversation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { conversationId: number; content: string }): Promise<MessageResponse> => {
      const { data } = await axiosInstance.post<MessageResponse>(
        `/messages/conversations/${payload.conversationId}/messages`,
        { content: payload.content },
      );
      return data;
    },
    onSuccess: (_message, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.messages.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.messages.conversation(variables.conversationId) });
    },
  });
}

export function useSendMessageToUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (payload: { userId: number; content: string }): Promise<MessageResponse> => {
      const { data } = await axiosInstance.post<MessageResponse>(
        `/messages/users/${payload.userId}/messages`,
        { content: payload.content },
      );
      return data;
    },
    onSuccess: (message) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.messages.conversations });
      queryClient.invalidateQueries({ queryKey: queryKeys.messages.conversation(message.conversationId) });
    },
  });
}

