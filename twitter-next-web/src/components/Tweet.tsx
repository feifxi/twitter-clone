'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Heart, MessageCircle, Repeat2, Share } from 'lucide-react';
import { toast } from 'sonner';
import { ConfirmModal } from '@/components/ConfirmModal';
import { ImageLightbox } from '@/components/ImageLightbox';
import { ReplyModal } from '@/components/ReplyModal';
import { TweetActionButton } from '@/components/TweetActionButton';
import { TweetActionsMenu } from '@/components/TweetActionsMenu';
import { TweetBody } from '@/components/TweetBody';
import { UserAvatar } from '@/components/UserAvatar';
import { useAuth } from '@/hooks/useAuth';
import { useDeleteTweet } from '@/hooks/useTweet';
import { useToggleLike, useToggleRetweet } from '@/hooks/useTweetActions';
import { displayNameOf, formatFullTime, formatRelativeTime } from '@/lib/format';
import { cn } from '@/lib/utils';
import { useUIStore } from '@/store/useUIStore';
import type { RetweetResponse, TweetResponse } from '@/types';

export function TweetSkeleton() {
  return (
    <div className="flex animate-pulse gap-3 border-b border-border px-4 py-3" aria-hidden="true">
      <div className="size-10 shrink-0 rounded-full bg-muted" />
      <div className="min-w-0 flex-1 space-y-2 py-1">
        <div className="h-3.5 w-40 rounded bg-muted" />
        <div className="h-3.5 w-full rounded bg-muted" />
        <div className="h-3.5 w-2/3 rounded bg-muted" />
      </div>
    </div>
  );
}

const stop = (e: React.MouseEvent) => e.stopPropagation();

function TweetMedia({ tweet, onOpenImage }: { tweet: TweetResponse; onOpenImage: (src: string) => void }) {
  if (!tweet.mediaUrl) return null;
  const src = tweet.mediaUrl;
  return (
    <div className="mt-3 overflow-hidden rounded-2xl border border-border">
      {tweet.mediaType === 'VIDEO' ? (
        <video src={src} controls playsInline preload="metadata" onClick={stop} className="max-h-[510px] w-full bg-black" />
      ) : (
        <button
          type="button"
          onClick={(e) => {
            stop(e);
            onOpenImage(src);
          }}
          className="block w-full cursor-zoom-in"
          aria-label="Open image"
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={src} alt="" loading="lazy" className="max-h-[510px] w-full object-cover" />
        </button>
      )}
    </div>
  );
}

interface TweetProps {
  tweet: TweetResponse;
  /** Set when the tweet is in a timeline because someone retweeted it ("X reposted"). */
  retweet?: RetweetResponse | null;
  /** The tweet detail page: larger text, full timestamp, no click-through. */
  variant?: 'feed' | 'detail';
}

export function Tweet({ tweet, retweet, variant = 'feed' }: TweetProps) {
  const router = useRouter();
  const toggleLike = useToggleLike();
  const toggleRetweet = useToggleRetweet();
  const deleteTweet = useDeleteTweet();
  const { user: me, isLoggedIn } = useAuth();
  const openSignInModal = useUIStore((s) => s.openSignInModal);
  const [replyOpen, setReplyOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  const { user: author } = tweet;
  const isDetail = variant === 'detail';
  const href = `/tweet/${tweet.id}`;

  /** Runs an action that needs a session, or asks to sign in. */
  const requireAuth = (action: () => void) => (e: React.MouseEvent) => {
    stop(e);
    if (!isLoggedIn) openSignInModal();
    else action();
  };

  const copyLink = () => {
    navigator.clipboard.writeText(`${window.location.origin}${href}`);
    toast.success('Copied to clipboard');
  };

  const actions = (
    <div className={cn('flex items-center justify-between', isDetail ? 'border-y border-border px-1 py-1' : '-ml-2 mt-1 max-w-[425px]')}>
      <TweetActionButton icon={<MessageCircle />} label="Reply" count={isDetail ? undefined : tweet.replyCount} onClick={requireAuth(() => setReplyOpen(true))} />
      <TweetActionButton
        icon={<Repeat2 />}
        label={tweet.isRetweeted ? 'Undo repost' : 'Repost'}
        tone="retweet"
        count={isDetail ? undefined : tweet.retweetCount}
        active={tweet.isRetweeted}
        onClick={requireAuth(() => toggleRetweet(tweet))}
      />
      <TweetActionButton
        icon={<Heart className={cn(tweet.isLiked && 'fill-current')} />}
        label={tweet.isLiked ? 'Unlike' : 'Like'}
        tone="like"
        count={isDetail ? undefined : tweet.likeCount}
        active={tweet.isLiked}
        onClick={requireAuth(() => toggleLike(tweet))}
      />
      <TweetActionButton icon={<Share />} label="Copy link" onClick={(e) => (stop(e), copyLink())} />
    </div>
  );

  const header = (
    <div className="flex items-start justify-between gap-2">
      <div className={cn('flex min-w-0 text-[15px] leading-5', isDetail ? 'flex-col' : 'items-center gap-1')}>
        <Link href={`/${author.username}`} onClick={stop} className="truncate font-bold hover:underline">
          {displayNameOf(author)}
        </Link>
        <span className="truncate text-muted-foreground">@{author.username}</span>
        {!isDetail && (
          <>
            <span className="text-muted-foreground" aria-hidden="true">·</span>
            <Link href={href} onClick={stop} className="shrink-0 text-muted-foreground hover:underline">
              {/* Relative time differs between server render and hydration ("45s" vs "46s"). */}
              <time dateTime={tweet.createdAt} title={formatFullTime(tweet.createdAt)} suppressHydrationWarning>
                {formatRelativeTime(tweet.createdAt)}
              </time>
            </Link>
          </>
        )}
      </div>
      <TweetActionsMenu isOwn={me?.id === author.id} onCopyLink={copyLink} onDelete={() => setConfirmDelete(true)} />
    </div>
  );

  return (
    <>
      <article
        onClick={isDetail ? undefined : () => router.push(href)}
        className={cn('border-b border-border px-4', isDetail ? 'pt-3' : 'cursor-pointer py-3 transition-colors hover:bg-foreground/[0.03]')}
      >
        {retweet && (
          <div className="mb-1 flex items-center gap-3 text-[13px] font-bold text-muted-foreground">
            <div className="flex w-10 justify-end">
              <Repeat2 className="size-4" />
            </div>
            <Link href={`/${retweet.retweeter.username}`} onClick={stop} className="truncate hover:underline">
              {me?.id === retweet.retweeter.id ? 'You' : displayNameOf(retweet.retweeter)} reposted
            </Link>
          </div>
        )}

        {isDetail ? (
          <>
            <div className="flex items-center gap-3">
              <UserAvatar user={author} link />
              <div className="min-w-0 flex-1">{header}</div>
            </div>
            {tweet.replyToUsername && <ReplyingTo username={tweet.replyToUsername} />}
            {tweet.content && <TweetBody content={tweet.content} className="mt-3 whitespace-pre-wrap break-words text-[17px] leading-6" />}
            <TweetMedia tweet={tweet} onOpenImage={setLightboxSrc} />
            <p className="my-4 text-[15px] text-muted-foreground">
              {/* Formatted in the viewer's time zone, which the server doesn't know. */}
            <time dateTime={tweet.createdAt} suppressHydrationWarning>{formatFullTime(tweet.createdAt)}</time>
            </p>
            {(tweet.retweetCount > 0 || tweet.likeCount > 0) && (
              <div className="flex gap-5 border-t border-border py-4 text-[15px]">
                {tweet.retweetCount > 0 && <Stat value={tweet.retweetCount} label="Reposts" />}
                {tweet.likeCount > 0 && <Stat value={tweet.likeCount} label="Likes" />}
              </div>
            )}
            {actions}
          </>
        ) : (
          <div className="flex gap-3">
            <UserAvatar user={author} link />
            <div className="min-w-0 flex-1">
              {header}
              {tweet.replyToUsername && <ReplyingTo username={tweet.replyToUsername} />}
              {tweet.content && <TweetBody content={tweet.content} className="whitespace-pre-wrap break-words text-[15px] leading-5" />}
              <TweetMedia tweet={tweet} onOpenImage={setLightboxSrc} />
              {actions}
            </div>
          </div>
        )}
      </article>

      {/* Siblings, not children: React bubbles clicks from portals to their parent component, which would open the tweet. */}
      <ReplyModal tweet={tweet} isOpen={replyOpen} onClose={() => setReplyOpen(false)} />
      <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />
      <ConfirmModal
        open={confirmDelete}
        title="Delete post?"
        description="This can't be undone and it will be removed from your profile, the timeline of any accounts that follow you, and from search results."
        confirmLabel="Delete"
        danger
        pending={deleteTweet.isPending}
        onConfirm={() => {
          deleteTweet.mutate(tweet);
          setConfirmDelete(false);
          if (isDetail) router.back();
        }}
        onClose={() => setConfirmDelete(false)}
      />
    </>
  );
}

function ReplyingTo({ username }: { username: string }) {
  return (
    <p className="text-[15px] text-muted-foreground">
      Replying to{' '}
      <Link href={`/${username}`} onClick={stop} className="text-primary hover:underline">
        @{username}
      </Link>
    </p>
  );
}

function Stat({ value, label }: { value: number; label: string }) {
  return (
    <span>
      <span className="font-bold">{value.toLocaleString()}</span> <span className="text-muted-foreground">{label}</span>
    </span>
  );
}
