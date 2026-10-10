'use client';

import { useEffect, useRef, useState } from 'react';
import dynamic from 'next/dynamic';
import { useTheme } from 'next-themes';
import { Image as ImageIcon, Smile, X } from 'lucide-react';
import { toast } from 'sonner';
import { Theme } from 'emoji-picker-react';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { UserAvatar } from '@/components/UserAvatar';
import { getErrorMessage } from '@/api/axiosInstance';
import { useAuth } from '@/hooks/useAuth';
import { useCreateTweet } from '@/hooks/useTweet';
import { cn } from '@/lib/utils';
import type { TweetResponse } from '@/types';

// Loaded on first open: the picker and its emoji data are large.
const EmojiPicker = dynamic(() => import('emoji-picker-react'), { ssr: false });

const MAX_CHARS = 280;
const MAX_MEDIA_BYTES = 100 * 1024 * 1024; // the API's MAX_MEDIA_BYTES
const ACCEPT = 'image/jpeg,image/png,image/gif,image/webp,video/mp4,video/webm';

/** X's character counter: a ring that fills up, with the remaining count in the last 20. */
function CharCountRing({ length }: { length: number }) {
  const remaining = MAX_CHARS - length;
  const size = remaining <= 20 ? 30 : 22;
  const radius = (size - 4) / 2;
  const circumference = 2 * Math.PI * radius;
  const progress = Math.min(length / MAX_CHARS, 1);
  const color = remaining <= 0 ? 'stroke-destructive' : remaining <= 20 ? 'stroke-amber-400' : 'stroke-primary';

  return (
    <span className="relative flex items-center justify-center" style={{ width: size, height: size }} aria-label={`${remaining} characters remaining`}>
      <svg width={size} height={size} className="-rotate-90">
        <circle cx={size / 2} cy={size / 2} r={radius} fill="none" strokeWidth={2.5} className="stroke-border" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          strokeWidth={2.5}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={circumference * (1 - progress)}
          className={cn(color, 'transition-[stroke-dashoffset] duration-100')}
        />
      </svg>
      {remaining <= 20 && (
        <span className={cn('absolute text-[12px] tabular-nums', remaining <= 0 ? 'text-destructive' : 'text-muted-foreground')}>{remaining}</span>
      )}
    </span>
  );
}

/** The text with #hashtags in blue, drawn under a transparent textarea. */
function Highlighted({ text }: { text: string }) {
  return (
    <>
      {text.split(/(#\w+)/g).map((part, i) =>
        /^#\w+$/.test(part) ? (
          <span key={i} className="text-primary">
            {part}
          </span>
        ) : (
          part
        ),
      )}
      {/* Keep the layer as tall as the textarea when the text ends in a newline. */}
      {text.endsWith('\n') && <br />}
    </>
  );
}

interface CreateTweetProps {
  placeholder?: string;
  isReply?: boolean;
  replyToId?: number;
  onSuccess?: (tweet: TweetResponse) => void;
  autoFocus?: boolean;
  className?: string;
}

export function CreateTweet({ placeholder = 'What is happening?!', isReply = false, replyToId, onSuccess, autoFocus, className }: CreateTweetProps) {
  const { user } = useAuth();
  const { resolvedTheme } = useTheme();
  const createTweet = useCreateTweet();
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const [content, setContent] = useState('');
  const [media, setMedia] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [focused, setFocused] = useState(false);

  // Free the preview's object URL when it changes and on unmount.
  useEffect(() => () => {
    if (previewUrl) URL.revokeObjectURL(previewUrl);
  }, [previewUrl]);

  // Grow with the text.
  useEffect(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${el.scrollHeight}px`;
  }, [content]);

  if (!user) return null;

  const pickFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;
    if (file.size > MAX_MEDIA_BYTES) {
      toast.error('That file is too large (100 MB max).');
      return;
    }
    setMedia(file);
    setPreviewUrl(URL.createObjectURL(file));
  };

  const clearMedia = () => {
    setMedia(null);
    setPreviewUrl(null);
  };

  const insertEmoji = (emoji: string) => {
    const el = textareaRef.current;
    const start = el?.selectionStart ?? content.length;
    const end = el?.selectionEnd ?? content.length;
    const next = (content.slice(0, start) + emoji + content.slice(end)).slice(0, MAX_CHARS);
    setContent(next);
    requestAnimationFrame(() => {
      el?.focus();
      el?.setSelectionRange(start + emoji.length, start + emoji.length);
    });
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!content.trim() && !media) return;
    try {
      const tweet = await createTweet.mutateAsync({ content: content.trim(), media: media ?? undefined, parentId: replyToId });
      setContent('');
      clearMedia();
      setFocused(false);
      toast.success(isReply ? 'Your reply was sent.' : 'Your post was sent.');
      onSuccess?.(tweet);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const expanded = focused || isReply || content.length > 0 || !!media;
  const canPost = (content.trim().length > 0 || !!media) && content.length <= MAX_CHARS && !createTweet.isPending;
  const textClass = 'whitespace-pre-wrap break-words text-[20px] leading-7';

  return (
    <form onSubmit={submit} className={cn('flex gap-3 border-b border-border px-4 py-3', className)}>
      <UserAvatar user={user} />
      <div className="min-w-0 flex-1">
        <div className="relative">
          <div aria-hidden="true" className={cn(textClass, 'pointer-events-none absolute inset-0 py-1')}>
            <Highlighted text={content} />
          </div>
          <textarea
            ref={textareaRef}
            value={content}
            onChange={(e) => setContent(e.target.value.slice(0, MAX_CHARS))}
            onFocus={() => setFocused(true)}
            placeholder={placeholder}
            aria-label={placeholder}
            autoFocus={autoFocus}
            rows={expanded ? 2 : 1}
            className={cn(textClass, 'relative block w-full resize-none overflow-hidden bg-transparent py-1 text-transparent caret-foreground outline-none placeholder:text-muted-foreground')}
          />
        </div>

        {previewUrl && (
          <div className="relative mt-3">
            {media?.type.startsWith('video/') ? (
              <video src={previewUrl} controls playsInline className="max-h-[300px] w-full rounded-2xl border border-border bg-black" />
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={previewUrl} alt="Attached media" className="max-h-[300px] w-full rounded-2xl border border-border object-cover" />
            )}
            <button
              type="button"
              onClick={clearMedia}
              aria-label="Remove media"
              className="absolute right-2 top-2 flex size-8 cursor-pointer items-center justify-center rounded-full bg-black/70 text-white backdrop-blur-sm hover:bg-black/90"
            >
              <X className="size-[18px]" />
            </button>
          </div>
        )}

        <div className={cn('mt-3 flex items-center justify-between', expanded && 'border-t border-border pt-3')}>
          <div className="-ml-2 flex items-center text-primary">
            <input ref={fileRef} type="file" accept={ACCEPT} className="hidden" onChange={pickFile} />
            <button type="button" onClick={() => fileRef.current?.click()} aria-label="Add photo or video" className="cursor-pointer rounded-full p-2 transition-colors hover:bg-primary/10">
              <ImageIcon className="size-5" />
            </button>
            <Popover>
              <PopoverTrigger asChild>
                <button type="button" aria-label="Add emoji" className="cursor-pointer rounded-full p-2 transition-colors hover:bg-primary/10">
                  <Smile className="size-5" />
                </button>
              </PopoverTrigger>
              <PopoverContent className="w-auto overflow-hidden p-0">
                <EmojiPicker
                  onEmojiClick={(data) => insertEmoji(data.emoji)}
                  theme={resolvedTheme === 'dark' ? Theme.DARK : Theme.LIGHT}
                  previewConfig={{ showPreview: false }}
                  height={400}
                  width={320}
                  lazyLoadEmojis
                />
              </PopoverContent>
            </Popover>
          </div>
          <div className="flex items-center gap-3">
            {content.length > 0 && <CharCountRing length={content.length} />}
            <button
              type="submit"
              disabled={!canPost}
              className="h-9 cursor-pointer rounded-full bg-primary px-4 text-[15px] font-bold text-primary-foreground transition-opacity hover:opacity-90 disabled:cursor-default disabled:opacity-50"
            >
              {createTweet.isPending ? 'Posting…' : isReply ? 'Reply' : 'Post'}
            </button>
          </div>
        </div>
      </div>
    </form>
  );
}
