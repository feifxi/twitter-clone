'use client';

import { Suspense } from 'react';
import { useSearchParams, useRouter, usePathname } from 'next/navigation';
import { FeedList } from '@/components/FeedList';
import { CreateTweet } from '@/components/CreateTweet';
import { useGlobalFeed, useFollowingFeed } from '@/hooks/useFeed';
import { useAuth } from '@/hooks/useAuth';

function HomeContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const { isLoggedIn } = useAuth();
  
  const currentTab = searchParams.get('tab') === 'following' ? 'following' : 'for-you';

  const globalFeed = useGlobalFeed(currentTab === 'for-you');
  const followingFeed = useFollowingFeed(isLoggedIn && currentTab === 'following');

  const feed = currentTab === 'for-you' ? globalFeed : followingFeed;
  const tweets = feed.data?.pages.flatMap((p) => p.items) ?? [];

  const handleTabChange = (newTab: 'for-you' | 'following') => {
    // Only update if changed
    if (newTab !== currentTab) {
        const params = new URLSearchParams(searchParams);
        params.set('tab', newTab);
        router.replace(`${pathname}?${params.toString()}`);
    }
  };

  return (
    <div className="min-h-screen">
      {/* Tab bar: flattened, blur */}
      <div className="sticky top-0 z-30 bg-background/80 backdrop-blur-md border-b border-border">
        <div className="flex">
          <button
            type="button"
            onClick={() => handleTabChange('for-you')}
            className={`flex-1 py-4 text-[15px] font-bold transition-colors hover:bg-card/50 cursor-pointer ${
              currentTab === 'for-you'
                ? 'text-foreground border-b-2 border-primary'
                : 'text-muted-foreground'
            }`}
          > 
            For you
          </button>
          <button
            type="button"
            onClick={() => handleTabChange('following')}
            disabled={!isLoggedIn}
            className={`flex-1 py-4 text-[15px] font-bold transition-colors hover:bg-card/50 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed ${
              currentTab === 'following'
                ? 'text-foreground border-b-2 border-primary'
                : 'text-muted-foreground'
            }`}
          >
            Following
          </button>
        </div>
      </div>
      {currentTab === 'following' && !isLoggedIn && (
        <div className="p-4 text-center text-muted-foreground text-[15px]">
          Sign in to see tweets from people you follow.
        </div>
      )}
      <div className="hidden sm:block border-b border-border">
        {isLoggedIn && <CreateTweet />}
      </div>
      {(currentTab !== 'following' || isLoggedIn) && (
        <FeedList
          tweets={tweets}
          isLoading={feed.isLoading}
          isFetchingNextPage={feed.isFetchingNextPage}
          hasNextPage={feed.hasNextPage}
          fetchNextPage={feed.fetchNextPage}
        />
      )}
    </div>
  );
}

export function Home() {
  return (
    <Suspense fallback={<div className="p-4 text-center text-muted-foreground">Loading...</div>}>
      <HomeContent />
    </Suspense>
  );
}
