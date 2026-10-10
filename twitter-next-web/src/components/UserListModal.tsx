import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog';
import { useUserFollowers, useUserFollowing } from '@/hooks/useProfile';
import { FollowButton } from '@/components/FollowButton';
import Link from 'next/link';
import { useInView } from 'react-intersection-observer';
import { useEffect } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { UserAvatar } from '@/components/UserAvatar';

interface UserListModalProps {
  userId: number | null;
  type: 'followers' | 'following' | null;
  isOpen: boolean;
  onClose: () => void;
}

export function UserListModal({ userId, type, isOpen, onClose }: UserListModalProps) {
  const { ref, inView } = useInView();
  const { user: currentUser } = useAuth();

  const isFollowers = type === 'followers';
  const queryHook = isFollowers ? useUserFollowers : useUserFollowing;
  
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = queryHook(userId);

  useEffect(() => {
    if (inView && hasNextPage) {
      fetchNextPage();
    }
  }, [inView, hasNextPage, fetchNextPage]);

  const users = data?.pages.flatMap((page) => page.items) ?? [];

  if (!type) return null;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="sm:max-w-[480px] p-0 gap-0 bg-background border-border text-foreground h-[600px] flex flex-col">
        <DialogHeader className="px-4 py-3 border-b border-border">
          <DialogTitle className="text-xl font-bold">
            {isFollowers ? 'Followers' : 'Following'}
          </DialogTitle>
          <DialogDescription className="sr-only">
            List of users {isFollowers ? 'following this account' : 'this account is following'}
          </DialogDescription>
        </DialogHeader>
        
        <div className="flex-1 overflow-y-auto min-h-0">
          {isLoading ? (
             <div className="p-4 text-center text-muted-foreground">Loading...</div>
          ) : users.length === 0 ? (
            <div className="p-8 text-center text-muted-foreground">
              <p className="font-bold text-lg mb-2">
                {isFollowers ? 'Looking empty here' : 'Be the first to follow'}
              </p>
              <p className="text-sm">
                {isFollowers 
                  ? 'This user has no followers yet.' 
                  : 'This user is not following anyone yet.'}
              </p>
            </div>
          ) : (
            <div className="flex flex-col">
              {users.map((user) => (
                <div key={user.id} className="flex items-center gap-3 px-4 py-3 hover:bg-card transition-colors">
                  <Link href={`/${user.username}`} onClick={onClose} className="shrink-0">
                    <UserAvatar user={user} />
                  </Link>
                  
                  <div className="flex-1 min-w-0">
                    <Link href={`/${user.username}`} onClick={onClose} className="block group">
                      <div className="font-bold text-foreground truncate group-hover:underline">
                        {user.displayName}
                      </div>
                      <div className="text-muted-foreground truncate">@{user.username}</div>
                    </Link>
                    {user.bio && <p className="text-foreground text-[14px] truncate mt-0.5">{user.bio}</p>}
                  </div>

                  {currentUser?.id !== user.id && (
                    <FollowButton userId={user.id} isFollowing={user.isFollowing} />
                  )}
                </div>
              ))}
              
              {/* Infinite scroll loader */}
              <div ref={ref} className="h-10 flex items-center justify-center p-4">
                {isFetchingNextPage && <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin" />}
              </div>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
