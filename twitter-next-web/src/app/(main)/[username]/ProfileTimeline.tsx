'use client';

import Link from 'next/link';
import { Play } from 'lucide-react';
import { FeedList } from '@/components/FeedList';
import { EmptyState, ErrorState } from '@/components/ListStates';
import { useAuth } from '@/hooks/useAuth';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import { useUserProfileByUsername, useUserTimeline } from '@/hooks/useProfile';
import type { ProfileTab } from '@/lib/query-keys';

export function ProfileTimeline({ username, tab }: { username: string; tab: ProfileTab }) {
  const { data: user } = useUserProfileByUsername(username);
  const { user: me } = useAuth();
  const isMe = !!user && me?.id === user.id;
  const handle = `@${username}`;
  const query = useUserTimeline(user?.id ?? null, tab, { enabled: tab !== 'likes' || isMe });

  if (tab === 'likes' && !isMe) {
    return <EmptyState title="Likes are private" body="Only the account owner can see what they've liked." />;
  }
  if (tab === 'media') return <MediaGrid query={query} handle={handle} />;

  const empty = {
    posts: { title: `${handle} hasn't posted`, body: 'When they do, their posts will show up here.' },
    replies: { title: `${handle} hasn't replied yet`, body: 'When they do, their replies will show up here.' },
    likes: { title: "You don't have any likes yet", body: 'Tap the heart on any post to show it some love. When you do, it’ll show up here.' },
  }[tab];
  return <FeedList query={query} empty={empty} />;
}

/** The Media tab: a 3-column grid of thumbnails linking to their tweets. */
function MediaGrid({ query, handle }: { query: ReturnType<typeof useUserTimeline>; handle: string }) {
  const sentinel = useInfiniteScroll(query);
  const tweets = query.data?.pages.flatMap((page) => page.items).filter((t) => t.mediaUrl) ?? [];

  if (query.isPending) {
    return (
      <div className="grid grid-cols-3 gap-0.5 p-0.5">
        {Array.from({ length: 9 }, (_, i) => <div key={i} className="aspect-square animate-pulse bg-muted" />)}
      </div>
    );
  }
  if (query.isError && tweets.length === 0) return <ErrorState onRetry={() => query.refetch()} />;
  if (tweets.length === 0) {
    return <EmptyState title={`${handle} hasn't posted media`} body="Once they do, those posts will show up here." />;
  }

  return (
    <div>
      <div className="grid grid-cols-3 gap-0.5 p-0.5">
        {tweets.map((tweet) => (
          <Link key={tweet.id} href={`/tweet/${tweet.id}`} className="relative aspect-square overflow-hidden bg-muted">
            {tweet.mediaType === 'VIDEO' ? (
              <>
                <video src={tweet.mediaUrl!} muted preload="metadata" className="size-full object-cover" />
                <Play className="absolute bottom-2 left-2 size-5 fill-white text-white drop-shadow" />
              </>
            ) : (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={tweet.mediaUrl!} alt="" loading="lazy" className="size-full object-cover transition-opacity hover:opacity-90" />
            )}
          </Link>
        ))}
      </div>
      <div ref={sentinel} />
    </div>
  );
}
