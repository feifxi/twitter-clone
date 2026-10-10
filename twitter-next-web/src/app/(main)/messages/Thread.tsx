'use client';

import { useRef, useState } from 'react';
import Link from 'next/link';
import { ArrowLeft, SendHorizontal } from 'lucide-react';
import { toast } from 'sonner';
import { getErrorMessage } from '@/api/axiosInstance';
import { ErrorState } from '@/components/ListStates';
import { UserAvatar } from '@/components/UserAvatar';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import { useConversationMessages, useSendMessageToConversation, useSendMessageToUser } from '@/hooks/useMessages';
import { displayNameOf } from '@/lib/format';
import type { UserResponse } from '@/types';
import { MessageBubble } from './MessageBubble';

interface ThreadProps {
  /** null for a new conversation: the first message creates it. */
  conversationId: number | null;
  peer: UserResponse;
  myId: number;
  onBack: () => void;
  onCreated: (conversationId: number) => void;
}

export function Thread({ conversationId, peer, myId, onBack, onCreated }: ThreadProps) {
  const query = useConversationMessages(conversationId);
  const sendToConversation = useSendMessageToConversation();
  const sendToUser = useSendMessageToUser();
  const [draft, setDraft] = useState('');
  const scroller = useRef<HTMLDivElement>(null);
  // In the reversed list the sentinel sits at the visual top: older messages load as you scroll up.
  const olderSentinel = useInfiniteScroll(query);
  const sending = sendToConversation.isPending || sendToUser.isPending;

  // Newest first: the container is flex-col-reverse, so the browser keeps the
  // view pinned to the latest message and older pages grow upwards on their own.
  const messages = (query.data?.pages.flatMap((page) => page.items) ?? []).sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime() || b.id - a.id,
  );

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    const content = draft.trim();
    if (!content || sending) return;
    try {
      if (conversationId) {
        await sendToConversation.mutateAsync({ conversationId, content });
      } else {
        const message = await sendToUser.mutateAsync({ userId: peer.id, content });
        onCreated(message.conversationId);
      }
      setDraft('');
      scroller.current?.scrollTo({ top: 0, behavior: 'smooth' }); // 0 = bottom in a reversed list
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  return (
    <section className="flex h-full min-w-0 flex-1 flex-col" aria-label={`Conversation with ${displayNameOf(peer)}`}>
      <header className="flex h-[53px] shrink-0 items-center gap-3 border-b border-border px-4">
        <button type="button" onClick={onBack} aria-label="Back to conversations" className="-ml-2 cursor-pointer rounded-full p-2 hover:bg-accent md:hidden">
          <ArrowLeft className="size-5" />
        </button>
        <Link href={`/${peer.username}`} className="flex min-w-0 items-center gap-3 hover:opacity-90">
          <UserAvatar user={peer} size="sm" />
          <span className="min-w-0">
            <span className="block truncate text-[17px] font-bold leading-5">{displayNameOf(peer)}</span>
            <span className="block truncate text-[13px] text-muted-foreground">@{peer.username}</span>
          </span>
        </Link>
      </header>

      <div ref={scroller} className="flex flex-1 flex-col-reverse gap-3 overflow-y-auto p-4">
        {messages.map((message) => (
          <MessageBubble key={message.id} message={message} mine={message.sender.id === myId} />
        ))}
        {conversationId && query.isPending && <p className="py-4 text-center text-[15px] text-muted-foreground">Loading messages…</p>}
        {query.isError && messages.length === 0 && <ErrorState onRetry={() => query.refetch()} />}
        {!conversationId && (
          <div className="m-auto max-w-xs py-10 text-center">
            <UserAvatar user={peer} size="lg" className="mx-auto" />
            <p className="mt-3 text-[17px] font-bold">{displayNameOf(peer)}</p>
            <p className="text-[15px] text-muted-foreground">Say hi to @{peer.username}. Your first message starts the conversation.</p>
          </div>
        )}
        {query.isFetchingNextPage && <p className="text-center text-[13px] text-muted-foreground">Loading older messages…</p>}
        <div ref={olderSentinel} className="h-px shrink-0" />
      </div>

      <form onSubmit={send} className="shrink-0 border-t border-border p-3">
        <div className="flex items-center gap-2 rounded-2xl bg-muted px-3 py-1">
          <input
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder="Start a new message"
            aria-label="Message"
            autoFocus
            maxLength={1000}
            className="h-10 min-w-0 flex-1 bg-transparent text-[15px] outline-none placeholder:text-muted-foreground"
          />
          <button
            type="submit"
            disabled={!draft.trim() || sending}
            aria-label="Send"
            className="cursor-pointer rounded-full p-2 text-primary transition-colors hover:bg-primary/10 disabled:cursor-default disabled:text-muted-foreground disabled:hover:bg-transparent"
          >
            <SendHorizontal className="size-5" />
          </button>
        </div>
      </form>
    </section>
  );
}
