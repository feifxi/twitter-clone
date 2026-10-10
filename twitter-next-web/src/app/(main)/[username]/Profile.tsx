'use client';

import { Suspense } from 'react';
import { FeedList } from '@/components/FeedList';
import { useUserProfileByUsername, useUserTimeline } from '@/hooks/useProfile';
import { FollowButton } from '@/components/FollowButton';
import { useAuth } from '@/hooks/useAuth';
import { EditProfileModal } from '@/components/EditProfileModal';
import { UserListModal } from '@/components/UserListModal';
import { ArrowLeft } from 'lucide-react';
import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { useState } from 'react';
import { UserAvatar } from '@/components/UserAvatar';

function ProfileContent({ username }: { username: string }) {
  const { data: user, isLoading: userLoading } = useUserProfileByUsername(username);
  const userId = user?.id ?? null;
  const feed = useUserTimeline(userId, 'posts');
  const { user: currentUser, isLoggedIn } = useAuth();
  const [showEditProfile, setShowEditProfile] = useState(false);
  const [listType, setListType] = useState<'followers' | 'following' | null>(null);
  
  const router = useRouter();
  const searchParams = useSearchParams();
  const pathname = usePathname();

  const currentTab = searchParams.get('tab') === 'media' ? 'media' : 'tweets';

  const tweets = feed.data?.pages.flatMap((p) => p.items) ?? [];
  const tweetsFiltered =
    currentTab === 'media'
      ? tweets.filter((t) => t.mediaUrl)
      : tweets;

  const handleTabChange = (newTab: 'tweets' | 'media') => {
    if (newTab !== currentTab) {
        const params = new URLSearchParams(searchParams);
        params.set('tab', newTab);
        router.replace(`${pathname}?${params.toString()}`);
    }
  };

  if (userLoading || !user) {
    return (
      <div className="p-4">
        <div className="h-32 w-32 rounded-full bg-card animate-pulse" />
        <div className="h-6 w-48 bg-card rounded mt-4 animate-pulse" />
        <div className="h-4 w-32 bg-card/60 rounded mt-2 animate-pulse" />
      </div>
    );
  }

  const isOwnProfile = currentUser?.id === user.id;

  return (
    <div className="min-h-screen">
      {/* Sticky Header */}
      <div className="sticky top-0 z-30 bg-background/60 backdrop-blur-md px-4 py-1 flex items-center gap-6 border-b border-border h-[53px]">
          <div 
              onClick={() => router.back()} 
              className="p-2 rounded-full hover:bg-card cursor-pointer transition-colors -ml-2"
          >
              <ArrowLeft className="w-5 h-5 text-foreground" />
          </div>
          <div className="flex flex-col">
              <h2 className="font-bold text-[20px] leading-6 text-foreground">{user.displayName}</h2>
              <span className="text-[13px] text-muted-foreground">{tweets.length} posts</span> 
          </div>
      </div>

      {/* Banner */}
      <div className="h-[200px] bg-secondary w-full relative overflow-hidden">
        {user.bannerUrl && (
            <img src={user.bannerUrl} alt="" className="w-full h-full object-cover" />
        )}
      </div>

      {/* Profile Info Container */}
      <div className="px-4 pb-4">
        <div className="flex justify-between items-start -mt-18 mb-4 relative z-10">
          <UserAvatar user={user} size="xl" className="border-4 border-background" />
          
          <div className="mt-20">
            {isLoggedIn && !isOwnProfile && (
                <FollowButton userId={user.id} isFollowing={user.isFollowing} />
            )}
            {isLoggedIn && isOwnProfile && (
                <button
                    type="button"
                    className="py-1.5 px-4 rounded-full border border-border text-foreground font-bold text-[15px] hover:bg-card transition-colors cursor-pointer"
                    onClick={() => setShowEditProfile(true)}
                >
                    Edit profile
                </button>
            )}
          </div>
        </div>

        <div className="flex flex-col">
          <h1 className="text-foreground font-black text-[20px] leading-6">
            {user.displayName}
          </h1>
          <p className="text-muted-foreground text-[15px] leading-5">@{user.username}</p>
          
          {user.bio && (
            <p className="text-foreground text-[15px] mt-3 mb-1 break-all whitespace-pre-wrap leading-5">{user.bio}</p>
          )}
          <div className="flex gap-5 mt-2 mb-1 text-[15px]">
            <button 
              className="hover:underline cursor-pointer flex gap-1 items-center"
              onClick={() => setListType('following')}
            >
              <strong className="text-foreground">{user.followingCount}</strong>
              <span className="text-muted-foreground">Following</span>
            </button>
            <button 
              className="hover:underline cursor-pointer flex gap-1 items-center"
              onClick={() => setListType('followers')}
            >
              <strong className="text-foreground">{user.followersCount}</strong>
              <span className="text-muted-foreground">Followers</span>
            </button>
          </div>
        </div>
      </div>
        
        <EditProfileModal user={user} isOpen={showEditProfile} onClose={() => setShowEditProfile(false)} />
        
        {user.id && (
          <UserListModal 
            userId={user.id} 
            type={listType} 
            isOpen={!!listType} 
            onClose={() => setListType(null)} 
          />
        )}

      <div className="flex border-b border-border">
        {(['tweets', 'media'] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => handleTabChange(t)}
            className={`flex-1 py-4 text-[15px] font-bold capitalize transition-colors hover:bg-card cursor-pointer relative ${
              currentTab === t
                ? 'text-foreground'
                : 'text-muted-foreground'
            }`}
          >
            {t}
            {currentTab === t && (
              <div className="absolute bottom-0 left-1/2 -translate-x-1/2 h-1 w-12 bg-primary rounded-full" />
            )}
          </button>
        ))}
      </div>

      <FeedList
          tweets={tweetsFiltered}
          isLoading={feed.isLoading}
          isFetchingNextPage={feed.isFetchingNextPage}
          hasNextPage={feed.hasNextPage}
          fetchNextPage={feed.fetchNextPage}
      />
    </div>
  );
}

/** The profile and its first posts arrive prefetched from the server page. */
export function Profile({ username }: { username: string }) {
    return (
        <Suspense fallback={<div className="p-4 text-center text-muted-foreground">Loading...</div>}>
            <ProfileContent username={username} />
        </Suspense>
    );
}
