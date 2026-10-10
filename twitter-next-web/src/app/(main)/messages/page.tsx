'use client';

import { Suspense, useEffect, useState } from 'react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { EmptyState } from '@/components/ListStates';
import { useAuth } from '@/hooks/useAuth';
import { useConversations } from '@/hooks/useMessages';
import { useUserProfile } from '@/hooks/useProfile';
import { cn } from '@/lib/utils';
import { useUIStore } from '@/store/useUIStore';
import { ConversationList } from './ConversationList';
import { NewMessageDialog } from './NewMessageDialog';
import { Thread } from './Thread';

const toId = (value: string | null) => (value && /^\d+$/.test(value) ? Number(value) : null);

/**
 * The open chat lives in the URL: ?c=<conversationId> for an existing one,
 * ?to=<userId> for a new one (it becomes ?c= after the first message). Links,
 * reloads and the back button all just work.
 */
function MessagesContent() {
  const { user: me } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const [composing, setComposing] = useState(false);

  const conversationId = toId(params.get('c'));
  const newPeerId = conversationId ? null : toId(params.get('to'));

  const conversations = useConversations();
  const loaded = conversations.data?.pages.flatMap((page) => page.items) ?? [];
  const open = conversationId ? loaded.find((c) => c.id === conversationId) : undefined;
  const newPeer = useUserProfile(newPeerId).data;

  // Messaging someone you already talk to opens that conversation instead.
  const existing = newPeerId ? loaded.find((c) => c.peer.id === newPeerId) : undefined;
  useEffect(() => {
    if (existing) router.replace(`${pathname}?c=${existing.id}`);
  }, [existing, pathname, router]);

  const go = (query: string) => router.push(query ? `${pathname}?${query}` : pathname);
  // A new chat waits for the conversation list, so the "reuse" redirect above
  // happens before it renders (not after the user has started typing).
  const peer = open?.peer ?? (conversations.isPending || existing ? undefined : newPeer);
  const chatOpen = !!(conversationId || newPeerId);

  return (
    <div className="flex h-screen">
      <div className={cn('w-full shrink-0 flex-col border-r border-border md:flex md:w-[320px] lg:w-[380px]', chatOpen ? 'hidden' : 'flex')}>
        <ConversationList activeId={conversationId} onOpen={(id) => go(`c=${id}`)} onNewMessage={() => setComposing(true)} />
      </div>

      <div className={cn('min-w-0 flex-1', chatOpen ? 'flex' : 'hidden md:flex')}>
        {chatOpen && peer && me ? (
          <Thread
            key={conversationId ?? `to-${newPeerId}`}
            conversationId={conversationId}
            peer={peer}
            myId={me.id}
            onBack={() => go('')}
            onCreated={(id) => router.replace(`${pathname}?c=${id}`)}
          />
        ) : chatOpen && !conversations.isPending && conversationId && !open ? (
          <EmptyState title="Conversation not found" body="It may have been removed, or the link is wrong." />
        ) : chatOpen ? (
          <p className="m-auto text-[15px] text-muted-foreground">Loading…</p>
        ) : (
          <div className="m-auto max-w-[400px] px-8">
            <EmptyState title="Select a message" body="Choose from your existing conversations, or start a new one." />
            <button type="button" onClick={() => setComposing(true)} className="mx-8 cursor-pointer rounded-full bg-primary px-8 py-4 text-[17px] font-bold text-primary-foreground hover:opacity-90">
              New message
            </button>
          </div>
        )}
      </div>

      <NewMessageDialog
        open={composing}
        onOpenChange={setComposing}
        myId={me?.id}
        onPick={(user) => {
          setComposing(false);
          go(`to=${user.id}`);
        }}
      />
    </div>
  );
}

export default function MessagesPage() {
  const { isLoggedIn } = useAuth();
  const openSignInModal = useUIStore((s) => s.openSignInModal);

  if (!isLoggedIn) {
    return (
      <div className="px-4">
        <EmptyState title="Welcome to your inbox!" body="Sign in to send private messages to people on Chanom Twitter." />
        <button type="button" onClick={openSignInModal} className="mx-8 cursor-pointer rounded-full bg-primary px-6 py-3 text-[15px] font-bold text-primary-foreground hover:opacity-90">
          Sign in
        </button>
      </div>
    );
  }
  return (
    <Suspense>
      <MessagesContent />
    </Suspense>
  );
}
