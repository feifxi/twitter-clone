'use client';

import { useEffect, useRef, useState } from 'react';
import ReactMarkdown, { type Components } from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Coffee, Send, Trash2 } from 'lucide-react';
import { ConfirmModal } from '@/components/ConfirmModal';
import { UserAvatar } from '@/components/UserAvatar';
import { useAssistantChat } from '@/hooks/useAssistantChat';
import { useAuth } from '@/hooks/useAuth';
import { useUIStore } from '@/store/useUIStore';

const markdown: Components = {
  p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
  a: ({ href, children }) => (
    <a href={href} target="_blank" rel="noopener noreferrer" className="text-primary hover:underline">
      {children}
    </a>
  ),
  code: ({ children }) => <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-sm">{children}</code>,
  pre: ({ children }) => <pre className="my-2 overflow-x-auto rounded-lg bg-muted p-3 text-sm">{children}</pre>,
  ul: ({ children }) => <ul className="mb-2 ml-4 list-disc">{children}</ul>,
  ol: ({ children }) => <ol className="mb-2 ml-4 list-decimal">{children}</ol>,
};

function BotAvatar({ size }: { size: 'sm' | 'md' }) {
  return (
    <div className={`flex shrink-0 items-center justify-center rounded-full bg-primary ${size === 'sm' ? 'size-8' : 'size-10'}`}>
      <Coffee className={`text-primary-foreground ${size === 'sm' ? 'size-5' : 'size-6'}`} />
    </div>
  );
}

export default function ChanomBotPage() {
  const { user, isLoggedIn } = useAuth();
  const { openSignInModal } = useUIStore();
  const { messages, streaming, send, clear } = useAssistantChat();
  const [input, setInput] = useState('');
  const [confirmClear, setConfirmClear] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);

  // Scroll only the message list (scrollIntoView would also scroll the page on phones)
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' });
  }, [messages]);

  const submit = () => {
    const query = input.trim();
    if (!query || streaming) return;
    if (!isLoggedIn) {
      openSignInModal();
      return;
    }
    setInput('');
    void send(query);
  };

  return (
    <div className="flex h-[calc(100dvh-60px)] max-w-full flex-col overflow-hidden sm:h-dvh">
      <header className="flex shrink-0 items-center justify-between border-b border-border bg-background/80 px-4 py-3 backdrop-blur-md">
        <div className="flex items-center gap-4">
          <BotAvatar size="sm" />
          <div>
            <h1 className="flex items-center gap-2 text-xl font-bold leading-tight">
              ChanomBot
              <span className="rounded-full bg-accent px-2 py-0.5 text-xs font-normal text-accent-foreground">Beta</span>
            </h1>
            <p className="text-xs leading-tight text-muted-foreground">Answers from what people are posting</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setConfirmClear(true)}
          disabled={streaming || messages.length === 1}
          className="flex cursor-pointer items-center gap-2 rounded-full px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive disabled:cursor-not-allowed disabled:opacity-50"
          aria-label="Clear chat"
        >
          <Trash2 className="size-4" />
          <span className="hidden sm:inline">Clear chat</span>
        </button>
      </header>

      <div ref={listRef} className="no-scrollbar flex-1 space-y-6 overflow-y-auto px-4 py-6">
        {messages.map((msg) => (
          <div key={msg.id} className="flex gap-4">
            {msg.role === 'assistant' || !user ? <BotAvatar size="md" /> : <UserAvatar user={user} />}
            <div className="flex min-w-0 flex-1 flex-col">
              <div className="mb-1 flex items-center gap-2 text-[15px]">
                <span className="font-bold">{msg.role === 'assistant' ? 'ChanomBot' : user?.displayName || 'You'}</span>
                {msg.role === 'user' && user && <span className="text-muted-foreground">@{user.username}</span>}
              </div>
              <div className="text-[15px] leading-relaxed">
                {msg.content === '' ? (
                  <div className="flex h-6 items-center gap-1" aria-label="ChanomBot is typing">
                    <div className="size-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.3s]" />
                    <div className="size-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:-0.15s]" />
                    <div className="size-1.5 animate-bounce rounded-full bg-muted-foreground" />
                  </div>
                ) : (
                  <ReactMarkdown remarkPlugins={[remarkGfm]} components={markdown}>
                    {msg.content}
                  </ReactMarkdown>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className="shrink-0 border-t border-border bg-background/90 px-4 pb-4 pt-2 backdrop-blur-md">
        <div className="mx-auto flex max-w-4xl items-end gap-2 rounded-2xl border border-border/50 bg-accent/30 p-2 transition-all focus-within:border-primary/50 focus-within:ring-1 focus-within:ring-primary/50">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                submit();
              }
            }}
            onInput={(e) => {
              const el = e.currentTarget;
              el.style.height = 'auto';
              el.style.height = `${Math.min(el.scrollHeight, 200)}px`;
            }}
            placeholder="Ask ChanomBot anything..."
            aria-label="Message ChanomBot"
            rows={1}
            className="max-h-48 min-h-[44px] flex-1 resize-none bg-transparent p-2 text-[15px] outline-none placeholder:text-muted-foreground"
          />
          <button
            type="button"
            onClick={submit}
            disabled={!input.trim() || streaming}
            className="flex size-9 shrink-0 cursor-pointer items-center justify-center rounded-full bg-primary text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="Send"
          >
            <Send className="size-4" />
          </button>
        </div>
        <p className="mt-3 text-center text-xs italic text-muted-foreground">
          This chat lasts until you leave the page, and ChanomBot remembers the last 30 messages.
        </p>
      </div>

      <ConfirmModal
        open={confirmClear}
        title="Clear the conversation?"
        description="Every message in this chat will be removed."
        confirmLabel="Clear"
        danger
        onConfirm={() => {
          clear();
          setConfirmClear(false);
        }}
        onClose={() => setConfirmClear(false)}
      />
    </div>
  );
}
