import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { refreshSession } from '@/api/axiosInstance';
import { useAuth } from '@/hooks/useAuth';
import { notificationQueryKey, unreadCountQueryKey } from './useNotifications';
import { conversationMessagesQueryKey, conversationsQueryKey } from './useMessages';
import type { MessageResponse } from '@/types/message';

const baseURL = process.env.NEXT_PUBLIC_API_URL;
const MAX_RETRY_DELAY_MS = 60_000;

/**
 * Subscribes to the API's single SSE stream (notifications + direct messages)
 * and refreshes the affected queries. Mount once, at layout level.
 */
export function useRealtimeEvents() {
  const queryClient = useQueryClient();
  const { isLoggedIn } = useAuth();

  useEffect(() => {
    if (!isLoggedIn) return;

    // Bursts of notifications are coalesced into one refetch.
    let refreshTimer: ReturnType<typeof setTimeout> | undefined;
    const refreshNotifications = () => {
      if (refreshTimer) return;
      refreshTimer = setTimeout(() => {
        refreshTimer = undefined;
        queryClient.invalidateQueries({ queryKey: unreadCountQueryKey });
        queryClient.invalidateQueries({ queryKey: notificationQueryKey });
      }, 250);
    };

    let source: EventSource | undefined;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let failures = 0;
    let connectedOnce = false;
    let stopped = false;

    const connect = () => {
      // The session cookies authenticate the stream.
      source = new EventSource(`${baseURL}/events`, { withCredentials: true });

      source.addEventListener('connected', () => {
        failures = 0;
        // Catch up on anything missed while disconnected.
        if (connectedOnce) {
          refreshNotifications();
          queryClient.invalidateQueries({ queryKey: conversationsQueryKey });
        }
        connectedOnce = true;
      });
      source.addEventListener('notification', refreshNotifications);
      source.addEventListener('dm', (event) => {
        try {
          const message = JSON.parse(event.data) as MessageResponse;
          queryClient.invalidateQueries({ queryKey: conversationsQueryKey });
          queryClient.invalidateQueries({ queryKey: conversationMessagesQueryKey(message.conversationId) });
        } catch {
          // Ignore malformed events; the next refetch corrects the view.
        }
      });

      source.onerror = () => {
        // On network blips the browser reconnects by itself (CONNECTING). It gives
        // up (CLOSED) on an HTTP error, usually an expired access token: renew the
        // session and reconnect, backing off while it keeps failing.
        if (stopped || source?.readyState !== EventSource.CLOSED) return;
        const delay = Math.min(1_000 * 2 ** failures++, MAX_RETRY_DELAY_MS);
        retryTimer = setTimeout(async () => {
          await refreshSession();
          if (!stopped) connect();
        }, delay);
      };
    };
    connect();

    return () => {
      stopped = true;
      source?.close();
      clearTimeout(retryTimer);
      clearTimeout(refreshTimer);
    };
  }, [isLoggedIn, queryClient]);
}
