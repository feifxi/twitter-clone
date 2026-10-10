'use client';

import { useRouter } from 'next/navigation';
import { X } from 'lucide-react';
import { CreateTweet } from '@/components/CreateTweet';
import { TweetBody } from '@/components/TweetBody';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';
import { UserAvatar } from '@/components/UserAvatar';
import { displayNameOf, formatRelativeTime } from '@/lib/format';
import type { TweetResponse } from '@/types';

interface ReplyModalProps {
  tweet: TweetResponse;
  isOpen: boolean;
  onClose: () => void;
}

/** Reply composer under a preview of the tweet, joined by a thread line. */
export function ReplyModal({ tweet, isOpen, onClose }: ReplyModalProps) {
  const router = useRouter();
  const { user } = tweet;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent showCloseButton={false} className="top-[5%] translate-y-0 gap-0 rounded-2xl p-0 sm:top-[10%] sm:max-w-[600px]">
        <DialogTitle className="sr-only">Reply</DialogTitle>
        <DialogDescription className="sr-only">Replying to @{user.username}</DialogDescription>
        <div className="flex h-[53px] items-center px-4">
          <button type="button" onClick={onClose} aria-label="Close" className="-ml-2 cursor-pointer rounded-full p-2 transition-colors hover:bg-accent">
            <X className="size-5" />
          </button>
        </div>

        <div className="flex gap-3 px-4">
          <div className="flex flex-col items-center">
            <UserAvatar user={user} />
            <div className="my-1 w-0.5 grow bg-border" />
          </div>
          <div className="min-w-0 flex-1 pb-4">
            <div className="flex items-center gap-1 text-[15px]">
              <span className="truncate font-bold">{displayNameOf(user)}</span>
              <span className="truncate text-muted-foreground">@{user.username}</span>
              <span className="shrink-0 text-muted-foreground">· {formatRelativeTime(tweet.createdAt)}</span>
            </div>
            {tweet.content && <TweetBody content={tweet.content} className="mt-0.5 whitespace-pre-wrap break-words text-[15px] leading-5" />}
            <p className="mt-3 text-[15px] text-muted-foreground">
              Replying to <span className="text-primary">@{user.username}</span>
            </p>
          </div>
        </div>

        <CreateTweet
          isReply
          autoFocus
          replyToId={tweet.id}
          placeholder="Post your reply"
          className="border-none pb-4 pt-1"
          onSuccess={() => {
            onClose();
            router.push(`/tweet/${tweet.id}`);
          }}
        />
      </DialogContent>
    </Dialog>
  );
}
