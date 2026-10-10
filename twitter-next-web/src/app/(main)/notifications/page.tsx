'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useNotifications, useMarkAllRead } from '@/hooks/useNotifications';
import { ArrowLeft, Heart, Repeat2, User, MessageCircle } from 'lucide-react';
import Link from 'next/link';
import { formatDistanceToNowStrict } from 'date-fns';
import { useInView } from 'react-intersection-observer';
import type { NotificationResponse } from '@/types';
import { UserAvatar } from '@/components/UserAvatar';

function NotificationItem({ notification }: { notification: NotificationResponse }) {
  const router = useRouter();
  const { actor, type, tweetId, tweetContent, tweetMediaUrl, originalTweetContent, originalTweetMediaUrl } = notification;

  let Icon = User;
  let iconColor = 'text-primary';
  let message = '';

  switch (type) {
    case 'LIKE':
        Icon = Heart;
        iconColor = 'text-like';
        message = 'liked your post';
        break;
    case 'RETWEET':
        Icon = Repeat2;
        iconColor = 'text-retweet';
        message = 'reposted your post';
        break;
    case 'FOLLOW':
        Icon = User;
        iconColor = 'text-primary';
        message = 'followed you';
        break;
    case 'REPLY':
        Icon = MessageCircle;
        iconColor = 'text-primary';
        message = 'replied to your post';
        break;
  }

  const handleClick = (e: React.MouseEvent) => {
    // If they clicked the actor link, don't double navigate
    if ((e.target as HTMLElement).closest('a')) return;
    
    if (type === 'FOLLOW') {
        router.push(`/${actor.username}`);
    } else if (tweetId) {
        router.push(`/tweet/${tweetId}`);
    }
  };

  return (
    <article 
        onClick={handleClick}
        className={`flex px-4 py-3 border-b border-border hover:bg-card/50 transition-colors cursor-pointer ${!notification.isRead ? 'bg-card/30' : ''}`}
    >
        {/* Left Column: Icon */}
        <div className="w-12 shrink-0 flex justify-end pr-3 pt-1">
            <Icon className={`w-7 h-7 ${iconColor} fill-current`} />
        </div>
        
        {/* Right Column: Content */}
        <div className="flex-1 min-w-0 flex flex-col gap-2 relative">
             
             {/* Avatar Row */}
             <div className="flex items-center">
                <Link href={`/${actor.username}`} onClick={(e) => e.stopPropagation()}>
                    <UserAvatar user={actor} size="sm" />
                </Link>
             </div>
             
             {/* Text Block */}
             <div className="text-[15px] text-foreground mt-0.5 leading-snug">
                {type === 'REPLY' ? (
                     <div className="flex justify-between items-start">
                         <div className="flex items-center gap-1 flex-wrap mb-0.5">
                            <Link href={`/${actor.username}`} className="font-bold hover:underline truncate max-w-[150px]" onClick={(e) => e.stopPropagation()}>
                                {actor.displayName}
                            </Link>
                            <span className="text-muted-foreground truncate max-w-[100px]">@{actor.username}</span>
                            <span className="text-muted-foreground">·</span>
                            <span className="text-muted-foreground hover:underline cursor-pointer">{formatDistanceToNowStrict(new Date(notification.createdAt))}</span>
                         </div>
                     </div>
                ) : (
                    <div className="flex items-center flex-wrap gap-x-1">
                        <Link href={`/${actor.username}`} className="font-bold hover:underline" onClick={(e) => e.stopPropagation()}>
                            {actor.displayName}
                        </Link>
                        <span>{message}</span>
                    </div>
                )}
             </div>
             
             {/* Snippet Context (The actual reply or the liked/retweeted post) */}
             {(tweetContent || tweetMediaUrl) && (
                <div className="text-foreground text-[15px] space-y-2 mt-0.5">
                    {type === 'REPLY' && <div className="text-muted-foreground text-[15px] mb-1">Replying to <span className="text-primary truncate min-w-0 inline-block align-bottom max-w-full hover:underline">@You</span></div>}
                    {tweetContent && (
                        <div className="whitespace-pre-wrap break-all leading-normal opacity-90">
                            {tweetContent}
                        </div>
                    )}
                    {tweetMediaUrl && (
                        <div className="relative w-[300px] h-[300px] rounded-2xl overflow-hidden border border-border mt-2">
                            {/* eslint-disable-next-line @next/next/no-img-element */}
                            <img src={tweetMediaUrl} alt="Media preview" className="object-cover w-full h-full" />
                        </div>
                    )}
                </div>
             )}

             {/* Original Tweet Context for Replies (Appears Below) */}
             {type === 'REPLY' && (originalTweetContent || originalTweetMediaUrl) && (
                 <div 
                    onClick={(e) => {
                        e.stopPropagation();
                        if (notification.originalTweetId) {
                            router.push(`/tweet/${notification.originalTweetId}`);
                        }
                    }}
                    className="mt-3 border border-border rounded-2xl p-3 flex flex-col gap-2 relative hover:bg-card/80 transition-colors cursor-pointer"
                 >
                    {originalTweetContent && (
                        <div className="text-[15px] text-muted-foreground whitespace-pre-wrap break-all line-clamp-3">
                            {originalTweetContent}
                        </div>
                    )}
                    
                    {originalTweetMediaUrl && (
                         <div className="relative w-full aspect-video max-h-[150px] rounded-xl overflow-hidden border border-border mt-1">
                             {/* eslint-disable-next-line @next/next/no-img-element */}
                             <img src={originalTweetMediaUrl} alt="Original media" className="object-cover w-full h-full" />
                         </div>
                    )}
                 </div>
             )}
        </div>
    </article>
  );
}

export default function NotificationsPage() {
  const router = useRouter();
  const { data, isLoading, fetchNextPage, hasNextPage, isFetchingNextPage } = useNotifications();
  const { mutate: markAllRead } = useMarkAllRead();
  const { ref, inView } = useInView();

  useEffect(() => {
    if (inView && hasNextPage) {
      fetchNextPage();
    }
  }, [inView, hasNextPage, fetchNextPage]);

  // Mark all as read on mount
  useEffect(() => {
      markAllRead();
  }, [markAllRead]);

  const notifications = data?.pages.flatMap((p) => p.items) ?? [];

  return (
    <div className="min-h-screen">
      <div className="sticky top-0 z-30 bg-background/80 backdrop-blur border-b border-border px-4 py-3 flex items-center gap-4">
        <div 
            onClick={() => router.back()} 
            className="p-2 rounded-full hover:bg-card cursor-pointer transition-colors -ml-2"
        >
            <ArrowLeft className="w-5 h-5 text-foreground" />
        </div>
        <h1 className="text-[20px] font-bold text-foreground">Notifications</h1>
      </div>

      {isLoading ? (
        <div className="p-4 text-muted-foreground">Loading...</div>
      ) : notifications.length === 0 ? (
        <div className="p-8 text-center text-muted-foreground">
            <p className="text-[30px] font-bold text-foreground mb-2">Nothing to see here — yet</p>
            <p>From likes to reposts and a whole lot more, this is where all the action happens.</p>
        </div>
      ) : (
        <div>
           {notifications.map((n) => (
             <NotificationItem key={n.id} notification={n} />
           ))}
           <div ref={ref} className="h-4" />
           {isFetchingNextPage && <div className="p-4 text-center text-muted-foreground">Loading more...</div>}
        </div>
      )}
    </div>
  );
}
