'use client';

import { Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { CreateTweet } from '@/components/CreateTweet';
import { FeedList } from '@/components/FeedList';
import { TabBar } from '@/components/TabBar';
import { useAuth } from '@/hooks/useAuth';
import { useFollowingFeed, useGlobalFeed } from '@/hooks/useFeed';

function HomeContent() {
  const { isLoggedIn } = useAuth();
  const following = useSearchParams().get('tab') === 'following' && isLoggedIn;

  const globalFeed = useGlobalFeed(!following);
  const followingFeed = useFollowingFeed(following);

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur-md">
        <TabBar
          replace
          tabs={[
            { label: 'For you', href: '/', active: !following },
            { label: 'Following', href: '/?tab=following', active: following, disabled: !isLoggedIn },
          ]}
        />
      </header>
      {isLoggedIn && (
        <div className="hidden border-b border-border sm:block">
          <CreateTweet />
        </div>
      )}
      {following ? (
        <FeedList
          query={followingFeed}
          empty={{ title: 'Welcome to your timeline', body: 'Follow some accounts and their posts and reposts will show up here.' }}
        />
      ) : (
        <FeedList query={globalFeed} empty={{ title: 'Nothing here yet', body: 'Be the first to post something.' }} />
      )}
    </div>
  );
}

export function Home() {
  return (
    <Suspense>
      <HomeContent />
    </Suspense>
  );
}
