import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { EventSourcePolyfill } from 'event-source-polyfill';
import { useAuthStore } from '@/store/useAuthStore';
import { notificationQueryKey, unreadCountQueryKey } from './useNotifications';
import { conversationMessagesQueryKey, conversationsQueryKey } from './useMessages';
import type { MessageResponse } from '@/types/message';

const baseURL = process.env.NEXT_PUBLIC_API_URL;

type SSEMessageEvent = { data: string };
type SSEListenerTarget = {
  addEventListener: (type: string, listener: (event: SSEMessageEvent) => void) => void;
};

/**
 * Subscribes to the API's single SSE stream (notifications + direct messages)
 * and refreshes the affected queries. Mount once, at layout level.
 */
export function useRealtimeEvents() {
  const queryClient = useQueryClient();
  const accessToken = useAuthStore((s) => s.accessToken);

  useEffect(() => {
    if (!accessToken) return;

    // Bursts of notifications are coalesced into one refetch.
    let refreshTimer: ReturnType<typeof setTimeout> | null = null;
    const refreshNotifications = () => {
      if (refreshTimer) return;
      refreshTimer = setTimeout(() => {
        refreshTimer = null;
        queryClient.invalidateQueries({ queryKey: unreadCountQueryKey });
        queryClient.invalidateQueries({ queryKey: notificationQueryKey });
      }, 250);
    };

    // The polyfill is needed only to send the Authorization header; cookie auth will remove it.
    const es = new EventSourcePolyfill(`${baseURL}/events`, {
      headers: { Authorization: `Bearer ${accessToken}` },
      heartbeatTimeout: 120_000,
    });
    const target = es as unknown as SSEListenerTarget;

    let connectedOnce = false;
    target.addEventListener('connected', () => {
      // Catch up on anything missed while disconnected.
      if (connectedOnce) {
        refreshNotifications();
        queryClient.invalidateQueries({ queryKey: conversationsQueryKey });
      }
      connectedOnce = true;
    });
    target.addEventListener('notification', refreshNotifications);
    target.addEventListener('dm', (event) => {
      try {
        const message = JSON.parse(event.data) as MessageResponse;
        queryClient.invalidateQueries({ queryKey: conversationsQueryKey });
        queryClient.invalidateQueries({ queryKey: conversationMessagesQueryKey(message.conversationId) });
      } catch {
        // Ignore malformed events; the next refetch corrects the view.
      }
    });

    return () => {
      es.close();
      if (refreshTimer) clearTimeout(refreshTimer);
    };
  }, [accessToken, queryClient]);
}
