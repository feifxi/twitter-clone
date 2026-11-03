'use client';

import { useEffect } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Heart, MessageCircle, Repeat2, User } from 'lucide-react';
import { EmptyState, ErrorState } from '@/components/ListStates';
import { PageHeader } from '@/components/PageHeader';
import { TweetSkeleton } from '@/components/Tweet';
import { TweetBody } from '@/components/TweetBody';
import { UserAvatar } from '@/components/UserAvatar';
import { useAuth } from '@/hooks/useAuth';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import { useMarkAllRead, useNotifications } from '@/hooks/useNotifications';
import { displayNameOf, formatRelativeTime } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { NotificationResponse, NotificationType, TweetPreviewResponse } from '@/types';

const KINDS: Record<NotificationType, { icon: typeof Heart; color: string; text: string }> = {
  LIKE: { icon: Heart, color: 'text-like', text: 'liked your post' },
  RETWEET: { icon: Repeat2, color: 'text-retweet', text: 'reposted your post' },
  FOLLOW: { icon: User, color: 'text-primary', text: 'followed you' },
  REPLY: { icon: MessageCircle, color: 'text-primary', text: 'replied to your post' },
};

const stop = (e: React.MouseEvent) => e.stopPropagation();

/** A still preview of a post's media; the post itself is one click away. */
function MediaThumb({ tweet, className }: { tweet: TweetPreviewResponse; className?: string }) {
  if (!tweet.mediaUrl) return null;
  const thumb = cn('mt-2 max-h-[200px] rounded-2xl border border-border object-cover', className);
  return tweet.mediaType === 'VIDEO' ? (
    <video src={tweet.mediaUrl} muted playsInline preload="metadata" className={thumb} />
  ) : (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={tweet.mediaUrl} alt="" loading="lazy" className={thumb} />
  );
}

/** A reply reads like a tweet; likes, reposts and follows are an icon row plus the post they're about. */
function NotificationItem({ notification, myUsername }: { notification: NotificationResponse; myUsername?: string }) {
  const router = useRouter();
  const { actor, type, tweet, originalTweet } = notification;
  const kind = KINDS[type];
  const href = type === 'FOLLOW' || !tweet ? `/${actor.username}` : `/tweet/${tweet.id}`;
  const Icon = kind.icon;

  return (
    <article
      onClick={() => router.push(href)}
      className={cn(
        'relative flex cursor-pointer gap-3 border-b border-border px-4 py-3 transition-colors hover:bg-foreground/[0.03]',
        !notification.isRead && 'bg-primary/[0.06]',
      )}
    >
      {!notification.isRead && <span className="absolute left-1.5 top-5 size-1.5 rounded-full bg-primary" aria-label="Unread" />}

      {type === 'REPLY' ? (
        <>
          <UserAvatar user={actor} link />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1 text-[15px]">
              <Link href={`/${actor.username}`} onClick={stop} className="truncate font-bold hover:underline">
                {displayNameOf(actor)}
              </Link>
              <span className="truncate text-muted-foreground">@{actor.username}</span>
              <span className="text-muted-foreground">·</span>
              <time dateTime={notification.createdAt} className="shrink-0 text-muted-foreground" suppressHydrationWarning>
                {formatRelativeTime(notification.createdAt)}
              </time>
            </div>
            <p className="text-[15px] text-muted-foreground">
              Replying to <span className="text-primary">@{myUsername}</span>
            </p>
            {tweet?.content && <TweetBody content={tweet.content} className="mt-1 whitespace-pre-wrap break-words text-[15px] leading-5" />}
            {tweet && <MediaThumb tweet={tweet} />}
            {originalTweet && (
              <Link
                href={`/tweet/${originalTweet.id}`}
                onClick={stop}
                className="mt-3 block rounded-2xl border border-border p-3 text-[15px] text-muted-foreground hover:bg-foreground/[0.03]"
              >
                {originalTweet.content && <span className="line-clamp-3 whitespace-pre-wrap">{originalTweet.content}</span>}
                <MediaThumb tweet={originalTweet} className={cn('max-h-[120px]', !originalTweet.content && 'mt-0')} />
              </Link>
            )}
          </div>
        </>
      ) : (
        <>
          <div className="flex w-10 shrink-0 justify-end">
            <Icon className={cn('size-7', kind.color, type !== 'FOLLOW' && 'fill-current')} />
          </div>
          <div className="min-w-0 flex-1">
            <UserAvatar user={actor} size="sm" link />
            <p className="mt-2 text-[15px]">
              <Link href={`/${actor.username}`} onClick={stop} className="font-bold hover:underline">
                {displayNameOf(actor)}
              </Link>{' '}
              {kind.text}
              <time dateTime={notification.createdAt} className="text-muted-foreground" suppressHydrationWarning>
                {' · '}
                {formatRelativeTime(notification.createdAt)}
              </time>
            </p>
            {tweet?.content && <p className="mt-2 line-clamp-4 whitespace-pre-wrap break-words text-[15px] text-muted-foreground">{tweet.content}</p>}
            {tweet && <MediaThumb tweet={tweet} />}
          </div>
        </>
      )}
    </article>
  );
}

export default function NotificationsPage() {
  const { user, isLoggedIn } = useAuth();
  const query = useNotifications();
  const { mutate: markAllRead } = useMarkAllRead();
  const sentinel = useInfiniteScroll(query);
  const notifications = query.data?.pages.flatMap((page) => page.items) ?? [];

  // Opening the page counts as seeing everything (the badge clears; highlights stay for this visit).
  useEffect(() => {
    if (isLoggedIn) markAllRead();
  }, [isLoggedIn, markAllRead]);

  return (
    <div className="min-h-screen">
      <PageHeader title="Notifications" />
      {!isLoggedIn ? (
        <EmptyState title="Sign in to see notifications" body="Likes, reposts, replies and new followers show up here." />
      ) : query.isPending ? (
        Array.from({ length: 4 }, (_, i) => <TweetSkeleton key={i} />)
      ) : query.isError && notifications.length === 0 ? (
        <ErrorState onRetry={() => query.refetch()} />
      ) : notifications.length === 0 ? (
        <EmptyState title="Nothing to see here — yet" body="From likes to reposts and a whole lot more, this is where all the action happens." />
      ) : (
        <div>
          {notifications.map((n) => (
            <NotificationItem key={n.id} notification={n} myUsername={user?.username} />
          ))}
          <div ref={sentinel} />
          {query.isFetchingNextPage && <TweetSkeleton />}
        </div>
      )}
    </div>
  );
}
