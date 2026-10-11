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
import type { NotificationResponse, NotificationType } from '@/types';

const KINDS: Record<NotificationType, { icon: typeof Heart; color: string; text: string }> = {
  LIKE: { icon: Heart, color: 'text-like', text: 'liked your post' },
  RETWEET: { icon: Repeat2, color: 'text-retweet', text: 'reposted your post' },
  FOLLOW: { icon: User, color: 'text-primary', text: 'followed you' },
  REPLY: { icon: MessageCircle, color: 'text-primary', text: 'replied to your post' },
};

const stop = (e: React.MouseEvent) => e.stopPropagation();

function MediaThumb({ src }: { src: string }) {
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={src} alt="" loading="lazy" className="mt-2 max-h-[200px] rounded-2xl border border-border object-cover" />;
}

/** A reply reads like a tweet; likes, reposts and follows are an icon row plus the post they're about. */
function NotificationItem({ notification, myUsername }: { notification: NotificationResponse; myUsername?: string }) {
  const router = useRouter();
  const { actor, type, tweetId, tweetContent, tweetMediaUrl } = notification;
  const kind = KINDS[type];
  const href = type === 'FOLLOW' || !tweetId ? `/${actor.username}` : `/tweet/${tweetId}`;
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
            {tweetContent && <TweetBody content={tweetContent} className="mt-1 whitespace-pre-wrap break-words text-[15px] leading-5" />}
            {tweetMediaUrl && <MediaThumb src={tweetMediaUrl} />}
            {notification.originalTweetContent && (
              <Link
                href={`/tweet/${notification.originalTweetId}`}
                onClick={stop}
                className="mt-3 block rounded-2xl border border-border p-3 text-[15px] text-muted-foreground hover:bg-foreground/[0.03]"
              >
                <span className="line-clamp-3 whitespace-pre-wrap">{notification.originalTweetContent}</span>
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
            {tweetContent && <p className="mt-2 line-clamp-4 whitespace-pre-wrap break-words text-[15px] text-muted-foreground">{tweetContent}</p>}
            {tweetMediaUrl && <MediaThumb src={tweetMediaUrl} />}
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
