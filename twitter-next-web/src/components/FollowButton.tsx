'use client';

import { Button } from '@/components/ui/button';
import { useToggleFollow } from '@/hooks/useProfile';
import { useAuth } from '@/hooks/useAuth';
import { useUIStore } from '@/store/useUIStore';

interface FollowButtonProps {
  userId: number;
  /** From the cache: flips immediately when clicked. */
  isFollowing: boolean;
}

export function FollowButton({ userId, isFollowing }: FollowButtonProps) {
  const toggleFollow = useToggleFollow();
  const { isLoggedIn } = useAuth();
  const openSignInModal = useUIStore((s) => s.openSignInModal);

  const handleFollow = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault(); // the button can sit inside a link (UserRow)
    if (!isLoggedIn) {
      openSignInModal();
      return;
    }
    toggleFollow({ id: userId, isFollowing });
  };

  return (
    <Button
      size="sm"
      variant={isFollowing ? "outline" : "secondary"}
      onClick={handleFollow}
      className={`rounded-full font-bold h-8 px-4 transition-colors cursor-pointer ${
        isFollowing
          ? 'hover:text-destructive hover:bg-destructive/10 border-border text-foreground hover:border-destructive/30'
          : 'bg-primary text-primary-foreground hover:bg-primary/90'
      }`}
    >
      {isFollowing ? 'Following' : 'Follow'}
    </Button>
  );
}
