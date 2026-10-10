'use client';

import { useRef, useState } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { useCreateTweet } from '@/hooks/useTweet';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Image as ImageIcon, X } from 'lucide-react';
import { useForm, useWatch } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { tweetRequestSchema, type TweetRequestInput } from '@/lib/validation';
import type { TweetResponse } from '@/types';

interface CreateTweetProps {
  placeholder?: string;
  isReply?: boolean;
  replyToId?: number;
  onSuccess?: (newTweet?: TweetResponse) => void;
  className?: string; // Allow custom styling wrapper
}

export function CreateTweet({
  placeholder = "What is happening?!",
  isReply = false,
  replyToId,
  onSuccess,
  className,
}: CreateTweetProps) {
  const { user } = useAuth();
  
  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { isValid }
  } = useForm<TweetRequestInput>({
    resolver: zodResolver(tweetRequestSchema),
    defaultValues: {
      content: '',
      parentId: replyToId,
    },
    mode: 'onChange',
  });

  const content = useWatch({ control, name: 'content' }) || '';
  
  const [media, setMedia] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const createMutation = useCreateTweet();

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setMedia(file);
      const url = URL.createObjectURL(file);
      setPreviewUrl(url);
    }
  };

  const clearMedia = () => {
    setMedia(null);
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    setPreviewUrl(null);
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const onSubmit = async (data: TweetRequestInput) => {
    if (!data.content?.trim() && !media) return;

    try {
      const newTweet = await createMutation.mutateAsync({
        content: data.content || '',
        media: media ?? undefined,
        parentId: replyToId,
      });
      reset();
      clearMedia();
      onSuccess?.(newTweet);
    } catch (error) {
      console.error('Failed to tweet:', error);
      // Ideally show a toast here
    }
  };

  const onFormSubmit: React.FormEventHandler<HTMLFormElement> = (e) => {
    e.preventDefault();
    void handleSubmit(onSubmit)(e);
  };

  // Helper to render text with highlighted hashtags
  const renderHighlightedText = (text: string) => {
    // Split by spaces but preserve them to maintain layout
    // We want to wrap words starting with # in a blue span
    return text.split(/(\s+)/).map((part, index) => {
      if (part.startsWith('#') && part.length > 1) {
        return (
          <span key={index} className="text-primary">
            {part}
          </span>
        );
      }
      return <span key={index}>{part}</span>;
    });
  };

  if (!user) return null;

  return (
    <div className={`flex gap-3 px-4 py-3 border-b border-border ${className || ''}`}>
      <div className="shrink-0">
        <Avatar className="w-10 h-10">
          <AvatarImage src={user.avatarUrl ?? undefined} alt={user.displayName || user.username} />
          <AvatarFallback>{(user.displayName || user.username)[0]}</AvatarFallback>
        </Avatar>
      </div>
      <form className="flex-1 min-w-0" onSubmit={onFormSubmit}>
        <div className="relative min-h-[52px]">
          {/* Highlighter Layer */}
          <div 
            className="absolute inset-0 whitespace-pre-wrap break-all text-[20px] font-normal text-foreground pointer-events-none"
            aria-hidden="true"
          >
            {renderHighlightedText(content)}
             {/* Add a zero-width space to ensure the height grows with the last newline if recently added */}
             {content.endsWith('\n') && <br />}
          </div>
          
          {/* Input Layer */}
          <textarea
            {...register('content')}
            placeholder={placeholder}
            className="w-full bg-transparent text-[20px] font-normal text-foreground placeholder-muted-foreground border-none outline-none resize-none overflow-hidden min-h-[52px] break-all"
            style={{ 
                // Color must be transparent so the highlighter shows through, 
                // BUT the caret color needs to be visible.
                // Standard approach: make text transparent but caret visible.
                 color: 'transparent', 
                 caretColor: 'var(--color-foreground)',
            }}
            // Auto-resize height
            onInput={(e) => {
                const target = e.target as HTMLTextAreaElement;
                target.style.height = 'auto'; 
                target.style.height = `${target.scrollHeight}px`;
            }}
            maxLength={280}
          />
        </div>

        {previewUrl && (
          <div className="relative mt-2 mb-2">
            {media?.type.startsWith('video/') ? (
              <video
                src={previewUrl}
                controls
                playsInline
                className="rounded-2xl max-h-[300px] border border-border bg-black"
              />
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={previewUrl}
                alt="Media preview"
                className="rounded-2xl max-h-[300px] object-cover border border-border"
              />
            )}
            <button
              type="button"
              onClick={clearMedia}
              className="absolute top-1 right-1 bg-black/75 rounded-full p-1 hover:bg-black/50 transition-colors"
            >
              <X className="w-5 h-5 text-white cursor-pointer" />
            </button>
          </div>
        )}

        <div className="flex items-center justify-between mt-2 border-t border-border pt-3">
          <div className="flex items-center gap-2 text-primary">
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="p-2 hover:bg-primary/10 rounded-full transition-colors"
              title="Media"
            >
              <ImageIcon className="w-5 h-5 cursor-pointer" />
            </button>
            <input
              type="file"
              ref={fileInputRef}
              className="hidden"
              accept="image/jpeg,image/png,image/gif,image/webp,video/mp4,video/webm"
              onChange={handleFileChange}
            />
          </div>
          <div className="flex items-center gap-4">
            {content.length > 0 && (
              <span className={`text-[13px] ${content.length >= 280 ? 'text-red-500' : 'text-muted-foreground'}`}>
                {content.length} / 280
              </span>
            )}
            <Button
              type="submit"
              disabled={(!content.trim() && !media) || createMutation.isPending || !isValid}
              className="rounded-full bg-primary hover:bg-primary/90 font-bold text-primary-foreground px-4 py-1.5 h-auto text-[15px] cursor-pointer"
            >
              {createMutation.isPending ? 'Posting...' : isReply ? 'Reply' : 'Post'}
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
}
