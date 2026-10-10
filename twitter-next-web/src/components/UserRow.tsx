'use client';

import Link from 'next/link';
import { FollowButton } from '@/components/FollowButton';
import { UserAvatar } from '@/components/UserAvatar';
import { useAuth } from '@/hooks/useAuth';
import { displayNameOf } from '@/lib/format';
import { cn } from '@/lib/utils';
import type { UserResponse } from '@/types';

interface UserRowProps {
  user: UserResponse;
  /** compact: no bio (sidebars); default: with bio (lists). */
  variant?: 'default' | 'compact';
  onNavigate?: () => void;
}

/** A user in a list: avatar, name, handle, optional bio, follow button. The whole row links to the profile. */
export function UserRow({ user, variant = 'default', onNavigate }: UserRowProps) {
  const { user: me } = useAuth();
  return (
    <div className="relative flex gap-3 px-4 py-3 transition-colors hover:bg-foreground/[0.03]">
      <UserAvatar user={user} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0 text-[15px] leading-5">
            {/* Stretched link: the row is clickable, the follow button stays separate. */}
            <Link href={`/${user.username}`} onClick={onNavigate} className="block truncate font-bold after:absolute after:inset-0 hover:underline">
              {displayNameOf(user)}
            </Link>
            <p className="truncate text-muted-foreground">@{user.username}</p>
          </div>
          {me?.id !== user.id && (
            <div className="relative shrink-0">
              <FollowButton userId={user.id} isFollowing={user.isFollowing} />
            </div>
          )}
        </div>
        {variant === 'default' && user.bio && <p className={cn('mt-1 line-clamp-2 text-[15px] leading-5')}>{user.bio}</p>}
      </div>
    </div>
  );
}

export function UserRowSkeleton() {
  return (
    <div className="flex animate-pulse items-center gap-3 px-4 py-3" aria-hidden="true">
      <div className="size-10 shrink-0 rounded-full bg-muted" />
      <div className="flex-1 space-y-2">
        <div className="h-3.5 w-32 rounded bg-muted" />
        <div className="h-3.5 w-20 rounded bg-muted" />
      </div>
      <div className="h-8 w-20 rounded-full bg-muted" />
    </div>
  );
}
