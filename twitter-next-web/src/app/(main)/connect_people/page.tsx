'use client';

import { useSuggestedUsers } from '@/hooks/useDiscovery';
import Link from 'next/link';
import { useAuth } from '@/hooks/useAuth';
import { FollowButton } from '@/components/FollowButton';

import { useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { UserAvatar } from '@/components/UserAvatar';

export default function ConnectPeoplePage() {
  const router = useRouter();
  const { data, isLoading } = useSuggestedUsers(20); // Fetch more users
  const users = data?.items ?? [];
  const { user: currentUser } = useAuth();

  return (
    <div className="min-h-screen">
      <div className="sticky top-0 z-30 bg-background/60 backdrop-blur-md border-b border-border px-4 py-3 flex items-center gap-4">
        <div 
            onClick={() => router.back()} 
            className="p-2 rounded-full hover:bg-card cursor-pointer transition-colors -ml-2"
        >
            <ArrowLeft className="w-5 h-5 text-foreground" />
        </div>
        <h1 className="text-xl font-bold">Connect</h1>
      </div>

      <div className="flex flex-col">
        {isLoading ? (
             <div className="p-4 space-y-4">
                {[1, 2, 3, 4, 5].map((i) => (
                    <div key={i} className="flex items-center gap-3">
                         <div className="w-10 h-10 rounded-full bg-border animate-pulse" />
                         <div className="flex-1 h-10 bg-border/50 rounded animate-pulse" />
                    </div>
                ))}
            </div>
        ) : users.length === 0 ? (
            <div className="p-4 text-center text-muted-foreground">No suggestions available.</div>
        ) : (
            users.map((u) => (
                <div key={u.id} className="flex items-center gap-3 px-4 py-3 hover:bg-card transition-colors cursor-pointer border-b border-border">
                        <Link href={`/${u.username}`} className="shrink-0">
                        <UserAvatar user={u} />
                    </Link>
                    <div className="flex-1 min-w-0">
                        <Link href={`/${u.username}`} className="font-bold text-foreground hover:underline block truncate">
                            {u.displayName}
                        </Link>
                        <div className="text-muted-foreground truncate">@{u.username}</div>
                        {u.bio && <p className="text-foreground text-[14px] mt-1 line-clamp-1">{u.bio}</p>}
                    </div>
                    {currentUser?.id !== u.id && (
                        <FollowButton userId={u.id} isFollowing={u.isFollowing} />
                    )}
                </div>
            ))
        )}
      </div>
    </div>
  );
}
