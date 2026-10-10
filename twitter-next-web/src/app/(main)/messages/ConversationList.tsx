'use client';

import { useState } from 'react';
import { MailPlus, Search } from 'lucide-react';
import { EmptyState, ErrorState } from '@/components/ListStates';
import { UserRowSkeleton } from '@/components/UserRow';
import { UserAvatar } from '@/components/UserAvatar';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import { useConversations } from '@/hooks/useMessages';
import { displayNameOf, formatRelativeTime } from '@/lib/format';
import { cn } from '@/lib/utils';

interface ConversationListProps {
  activeId: number | null;
  onOpen: (conversationId: number) => void;
  onNewMessage: () => void;
}

export function ConversationList({ activeId, onOpen, onNewMessage }: ConversationListProps) {
  const query = useConversations();
  const sentinel = useInfiniteScroll(query);
  const [filter, setFilter] = useState('');
  const needle = filter.trim().toLowerCase();
  const conversations = (query.data?.pages.flatMap((page) => page.items) ?? []).filter(
    (c) => !needle || c.peer.username.toLowerCase().includes(needle) || (c.peer.displayName ?? '').toLowerCase().includes(needle),
  );

  return (
    <>
      <header className="flex h-[53px] shrink-0 items-center justify-between px-4">
        <h1 className="text-[20px] font-bold">Messages</h1>
        <button type="button" onClick={onNewMessage} aria-label="New message" className="cursor-pointer rounded-full p-2 hover:bg-accent">
          <MailPlus className="size-5" />
        </button>
      </header>
      <label className="mx-4 mb-2 flex items-center gap-3 rounded-full border border-transparent bg-muted px-4 py-2 focus-within:border-primary focus-within:bg-background">
        <Search className="size-4 text-muted-foreground" />
        <input
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          placeholder="Search Direct Messages"
          aria-label="Search Direct Messages"
          className="min-w-0 flex-1 bg-transparent text-[15px] outline-none placeholder:text-muted-foreground"
        />
      </label>

      <div className="flex-1 overflow-y-auto">
        {query.isPending ? (
          Array.from({ length: 5 }, (_, i) => <UserRowSkeleton key={i} />)
        ) : query.isError ? (
          <ErrorState onRetry={() => query.refetch()} />
        ) : conversations.length === 0 ? (
          needle ? (
            <p className="px-4 py-6 text-[15px] text-muted-foreground">No conversations match &quot;{filter}&quot;.</p>
          ) : (
            <div>
              <EmptyState title="Welcome to your inbox!" body="Drop a line, share a post and more with private conversations between you and others." />
              <button type="button" onClick={onNewMessage} className="mx-8 cursor-pointer rounded-full bg-primary px-6 py-3 text-[15px] font-bold text-primary-foreground hover:opacity-90">
                Write a message
              </button>
            </div>
          )
        ) : (
          <ul>
            {conversations.map((conversation) => (
              <li key={conversation.id}>
                <button
                  type="button"
                  onClick={() => onOpen(conversation.id)}
                  aria-current={conversation.id === activeId ? 'true' : undefined}
                  className={cn(
                    'flex w-full cursor-pointer gap-3 px-4 py-3 text-left transition-colors hover:bg-foreground/[0.03]',
                    conversation.id === activeId && 'border-r-2 border-primary bg-foreground/[0.04]',
                  )}
                >
                  <UserAvatar user={conversation.peer} />
                  <span className="min-w-0 flex-1">
                    <span className="flex items-baseline gap-1 text-[15px]">
                      <span className="truncate font-bold">{displayNameOf(conversation.peer)}</span>
                      <span className="truncate text-muted-foreground">@{conversation.peer.username}</span>
                      <span className="shrink-0 text-muted-foreground" suppressHydrationWarning>
                        · {formatRelativeTime(conversation.lastMessage.createdAt)}
                      </span>
                    </span>
                    <span className="block truncate text-[15px] text-muted-foreground">{conversation.lastMessage.content}</span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
        )}
        <div ref={sentinel} />
      </div>
    </>
  );
}
