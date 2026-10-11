'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { format } from 'date-fns';
import { CalendarDays, Link2, MapPin } from 'lucide-react';
import { EditProfileModal } from '@/components/EditProfileModal';
import { FollowButton } from '@/components/FollowButton';
import { ImageLightbox } from '@/components/ImageLightbox';
import { PageHeader } from '@/components/PageHeader';
import { TabBar } from '@/components/TabBar';
import { TweetBody } from '@/components/TweetBody';
import { UserAvatar } from '@/components/UserAvatar';
import { useAuth } from '@/hooks/useAuth';
import { useUserProfileByUsername } from '@/hooks/useProfile';
import { displayNameOf, formatCount } from '@/lib/format';
import { useUIStore } from '@/store/useUIStore';

export function ProfileHeader({ username }: { username: string }) {
  // Seeded by the server layout, so this is never pending on first render.
  const { data: user } = useUserProfileByUsername(username);
  const { user: me } = useAuth();
  const openSignInModal = useUIStore((s) => s.openSignInModal);
  const pathname = usePathname();
  const [editing, setEditing] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  if (!user) return null;
  const isMe = me?.id === user.id;
  const base = `/${user.username}`;
  const tabs = [
    { label: 'Posts', href: base, active: pathname === base },
    { label: 'Replies', href: `${base}/replies`, active: pathname === `${base}/replies` },
    { label: 'Media', href: `${base}/media`, active: pathname === `${base}/media` },
    // Likes are private: only the owner sees the tab.
    ...(isMe ? [{ label: 'Likes', href: `${base}/likes`, active: pathname === `${base}/likes` }] : []),
  ];

  return (
    <>
      <PageHeader title={displayNameOf(user)} subtitle={`@${user.username}`} back />

      <div className="relative aspect-[3/1] w-full bg-banner">
        {user.bannerUrl && (
          <button type="button" onClick={() => setLightboxSrc(user.bannerUrl)} className="size-full cursor-zoom-in" aria-label="Open banner">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={user.bannerUrl} alt="" className="size-full object-cover" />
          </button>
        )}
      </div>

      <div className="px-4 pb-3">
        <div className="flex items-start justify-between">
          <button
            type="button"
            onClick={() => user.avatarUrl && setLightboxSrc(user.avatarUrl)}
            className="-mt-[15%] rounded-full sm:-mt-[67px]"
            aria-label="Open profile photo"
            disabled={!user.avatarUrl}
          >
            <UserAvatar user={user} size="xl" className="size-[25vw] max-h-[134px] max-w-[134px] border-4 border-background sm:size-[134px]" />
          </button>
          <div className="pt-3">
            {isMe ? (
              <button
                type="button"
                onClick={() => setEditing(true)}
                className="h-9 cursor-pointer rounded-full border border-border px-4 text-[15px] font-bold transition-colors hover:bg-accent"
              >
                Edit profile
              </button>
            ) : me ? (
              <FollowButton userId={user.id} isFollowing={user.isFollowing} />
            ) : (
              <button
                type="button"
                onClick={openSignInModal}
                className="h-9 cursor-pointer rounded-full bg-foreground px-4 text-[15px] font-bold text-background hover:opacity-90"
              >
                Follow
              </button>
            )}
          </div>
        </div>

        <div className="mt-3">
          <h2 className="text-[20px] font-extrabold leading-6">{displayNameOf(user)}</h2>
          <p className="text-[15px] text-muted-foreground">@{user.username}</p>
        </div>

        {user.bio && <TweetBody content={user.bio} className="mt-3 whitespace-pre-wrap break-words text-[15px] leading-5" />}

        <div className="mt-3 flex flex-wrap gap-x-3 gap-y-1 text-[15px] text-muted-foreground">
          {user.location && (
            <span className="flex items-center gap-1">
              <MapPin className="size-[18px]" />
              {user.location}
            </span>
          )}
          {user.website && (
            <a href={user.website} target="_blank" rel="noopener noreferrer nofollow" className="flex items-center gap-1 text-primary hover:underline">
              <Link2 className="size-[18px] text-muted-foreground" />
              {user.website.replace(/^https?:\/\/(www\.)?/, '').replace(/\/$/, '')}
            </a>
          )}
          <span className="flex items-center gap-1">
            <CalendarDays className="size-[18px]" />
            Joined {format(new Date(user.createdAt), 'MMMM yyyy')}
          </span>
        </div>

        <div className="mt-3 flex gap-5 text-[15px]">
          <Link href={`${base}/following`} className="hover:underline">
            <span className="font-bold">{formatCount(user.followingCount)}</span> <span className="text-muted-foreground">Following</span>
          </Link>
          <Link href={`${base}/followers`} className="hover:underline">
            <span className="font-bold">{formatCount(user.followersCount)}</span>{' '}
            <span className="text-muted-foreground">{user.followersCount === 1 ? 'Follower' : 'Followers'}</span>
          </Link>
        </div>
      </div>

      <TabBar tabs={tabs} className="border-b border-border" />

      {isMe && <EditProfileModal user={user} isOpen={editing} onClose={() => setEditing(false)} />}
      <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />
    </>
  );
}
