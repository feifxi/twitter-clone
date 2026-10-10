'use client';

import { useState } from 'react';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Heart, MessageCircle, Repeat2, Share, MoreHorizontal } from 'lucide-react';
import type { TweetResponse } from '@/types';
import { useLikeTweet, useUnlikeTweet } from '@/hooks/useLike';
import { useRetweet, useUnretweet } from '@/hooks/useRetweet';
import { ReplyModal } from '@/components/ReplyModal';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { useAuth } from '@/hooks/useAuth';
import { useUIStore } from '@/store/useUIStore';
import { useDeleteTweet } from '@/hooks/useTweet';
import { useDebouncedToggle } from '@/hooks/useDebouncedToggle';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import { formatDistanceToNowStrict } from 'date-fns';
import { Trash2 } from 'lucide-react';
import { toast } from 'sonner';

function formatDate(iso: string) {
  try {
      return formatDistanceToNowStrict(new Date(iso))
        .replace(' seconds', 's')
        .replace(' second', 's')
        .replace(' minutes', 'm')
        .replace(' minute', 'm')
        .replace(' hours', 'h')
        .replace(' hour', 'h')
        .replace(' days', 'd')
  } catch {
      return '';
  }
}

export function TweetSkeleton() {
  return (
    <div className="flex gap-3 p-4 border-b border-border animate-pulse">
      <div className="w-10 h-10 rounded-full bg-secondary shrink-0" />
      <div className="flex-1 min-w-0 space-y-2">
        <div className="h-4 w-32 bg-secondary rounded" />
        <div className="h-16 bg-secondary/50 rounded" />
      </div>
    </div>
  );
}

interface TweetProps {
  tweet: TweetResponse;
  disableClick?: boolean;
}

export function Tweet({ tweet, disableClick }: TweetProps) {
  const router = useRouter();
  const likeMutation = useLikeTweet();
  const unlikeMutation = useUnlikeTweet();
  const retweetMutation = useRetweet();
  const unretweetMutation = useUnretweet();
  const deleteMutation = useDeleteTweet();
  const [showReplyModal, setShowReplyModal] = useState(false);
  const { user: currentUser, isLoggedIn } = useAuth();
  const openSignInModal = useUIStore((s) => s.openSignInModal);

  const displayTweet = tweet;
  const user = displayTweet.user;
  const isRetweet = !!tweet.retweetedBy;

  // Prevent bubbling for interactive elements
  const stopProp = (e: React.MouseEvent) => {
    e.stopPropagation();
  };
  
    const handleCardClick = () => {
    router.push(`/tweet/${displayTweet.id}`);
  };

  const {
      optimisticState: isLiked,
      toggle: toggleLike,
  } = useDebouncedToggle({
      initialState: displayTweet.isLiked,
      onMutate: (newState) => {
          if (newState) likeMutation.mutate(displayTweet.id);
          else unlikeMutation.mutate(displayTweet.id);
      },
  });

  const {
      optimisticState: isRetweeted,
      toggle: toggleRetweet,
  } = useDebouncedToggle({
      initialState: displayTweet.isRetweeted,
      onMutate: (newState) => {
          if (newState) retweetMutation.mutate(displayTweet.id);
          else unretweetMutation.mutate(displayTweet.id);
      },
  });

  // Calculate optimistic counts
  const likeCount = displayTweet.likeCount + (isLiked ? 1 : 0) - (displayTweet.isLiked ? 1 : 0);
  const retweetCount = displayTweet.retweetCount + (isRetweeted ? 1 : 0) - (displayTweet.isRetweeted ? 1 : 0);

  const handleLike = (e: React.MouseEvent) => {
    stopProp(e);
    if (!isLoggedIn) {
      openSignInModal();
      return;
    }
    toggleLike();
  };

  const handleReply = (e: React.MouseEvent) => {
    stopProp(e);
    if (!isLoggedIn) {
      openSignInModal();
      return;
    }
    setShowReplyModal(true);
  };

  const handleRetweet = (e: React.MouseEvent) => {
    stopProp(e);
    if (!isLoggedIn) {
      openSignInModal();
      return;
    }
    toggleRetweet();
  };

  const handleShare = (e: React.MouseEvent) => {
    stopProp(e);
    const url = `${window.location.origin}/tweet/${displayTweet.id}`;
    navigator.clipboard.writeText(url);
    toast.success('Copied to clipboard');
  };

  return (
    <article
        className={cn(
          "px-4 py-3 border-b border-border flex gap-3 relative",
          !disableClick && "hover:bg-card/50 transition-colors cursor-pointer"
        )}
        onClick={!disableClick ? handleCardClick : undefined}
    >
        {/* Main Link Wrapper (absolute to cover everything except buttons) */}
        {/* <Link href={`/tweet/${displayTweet.id}`} className="absolute inset-0 z-0" aria-hidden="true" /> */}

      {/* Left Column: Avatar */}
      <div className="shrink-0 z-10">
        <Link href={`/${user.username}`} onClick={stopProp}>
            <Avatar className="w-10 h-10 border border-border/50 hover:opacity-90 transition-opacity">
                <AvatarImage src={user.avatarUrl ?? undefined} alt={user.displayName || user.username} />
                <AvatarFallback>{(user.displayName || user.username)[0]}</AvatarFallback>
            </Avatar>
        </Link>
      </div>

      {/* Right Column: Content */}
      <div className="flex-1 min-w-0 z-10">
        {/* Context Header (Retweeted) */}
        {isRetweet && (
            <div className="flex items-center gap-2 text-muted-foreground text-[13px] font-bold mb-1 -ml-6">
                <div className="w-8 flex justify-end">
                    <Repeat2 className="w-4 h-4" />
                </div>
                <Link href={`/${tweet.retweetedBy!.username}`} className="hover:underline" onClick={stopProp}>
                    {tweet.retweetedBy!.displayName || tweet.retweetedBy!.username} reposted
                </Link>
            </div>
        )}

        {/* User Header */}
        <div className="flex items-center justify-between">
            <div className="flex items-center gap-1 text-[15px] truncate overflow-hidden">
                <Link href={`/${user.username}`} className="font-bold text-foreground hover:underline truncate" onClick={stopProp}>
                    {user.displayName}
                </Link>
                <span className="text-muted-foreground truncate">@{user.username}</span>
                <span className="text-muted-foreground">·</span>
                <span className="text-muted-foreground hover:underline">{formatDate(displayTweet.createdAt)}</span>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-full -mr-2" onClick={stopProp}>
                    <MoreHorizontal className="w-4 h-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-[200px] bg-background border-border text-foreground">
                 {currentUser?.id === user.id && (
                  <DropdownMenuItem 
                    onClick={(e) => {
                      stopProp(e);
                      deleteMutation.mutate(displayTweet.id);
                    }}
                    className="text-destructive focus:text-destructive focus:bg-destructive/10 cursor-pointer gap-2 font-bold"
                  >
                    <Trash2 className="w-4 h-4" />
                    Delete
                  </DropdownMenuItem>
                 )}
                 {/* Can add more generic options here like 'Not interested' etc for non-owners */}
                  {currentUser?.id !== user.id && (
                     <DropdownMenuItem 
                      className="text-foreground focus:bg-card cursor-pointer font-bold"
                      onClick={stopProp}
                    >
                      Not interested in this Post
                    </DropdownMenuItem>
                  )}
              </DropdownMenuContent>
            </DropdownMenu>
        </div>

        {/* Reply Context */}
        {displayTweet.replyToUsername && (
             <div className="text-muted-foreground text-[15px] mb-1">
                Replying to <Link href={`/${displayTweet.replyToUsername}`} className="text-primary hover:underline" onClick={stopProp}>@{displayTweet.replyToUsername}</Link>
             </div>
        )}

        {/* Tweet Content */}
        <div className="text-foreground text-[15px] whitespace-pre-wrap leading-6 break-all mb-2">
            {(displayTweet.content || '').split(/(\s+)/).map((part, index) => {
              if (part.startsWith('#') && part.length > 1) {
                return (
                  <Link 
                    key={index} 
                    href={`/search?q=${encodeURIComponent(part)}`}
                    className="text-primary hover:underline cursor-pointer"
                    onClick={stopProp}
                  >
                    {part}
                  </Link>
                );
              }
              return <span key={index}>{part}</span>;
            })}
        </div>

        {/* Media */}
        {displayTweet.mediaUrl && (
            <div className="rounded-2xl border border-border overflow-hidden mt-2 mb-2 max-h-[500px]">
                {displayTweet.mediaType === 'VIDEO' ? (
                  <video
                    src={displayTweet.mediaUrl}
                    controls
                    playsInline
                    preload="metadata"
                    onClick={stopProp}
                    className="w-full max-h-[500px] bg-black"
                  />
                ) : (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={displayTweet.mediaUrl} alt="Tweet media" className="w-full max-h-[500px] object-cover text-transparent" />
                )}
            </div>
        )}

        {/* Action Bar */}
        <div className="flex items-center justify-between max-w-[425px] mt-1 -ml-2">
            {/* Reply */}
            <Button 
                variant="ghost" 
                size="sm" 
                className="group flex items-center gap-1 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-full px-2 h-8 cursor-pointer" 
                onClick={handleReply}
            >
                <MessageCircle className="w-[18px] h-[18px] group-hover:text-primary transition-colors" />
                <span className="text-[13px] group-hover:text-primary transition-colors">{displayTweet.replyCount || ''}</span>
            </Button>
            


            {/* Retweet */}
            <Button
                variant="ghost"
                size="sm"
                className={cn(
                    "group flex items-center gap-1 text-muted-foreground hover:bg-green-500/10 rounded-full px-2 h-8 cursor-pointer",
                    isRetweeted ? "text-green-500" : "hover:text-green-500"
                )}
                onClick={handleRetweet}
            >
                <Repeat2 className="w-[18px] h-[18px] transition-colors" />
                <span className="text-[13px] transition-colors">{retweetCount || ''}</span>
            </Button>

            {/* Like */}
            <Button
                 variant="ghost"
                 size="sm"
                 className={cn(
                    "group flex items-center gap-1 text-muted-foreground hover:bg-pink-500/10 rounded-full px-2 h-8 cursor-pointer",
                    isLiked ? "text-pink-600" : "hover:text-pink-600"
                 )}
                 onClick={handleLike}
            >
                <Heart className={cn("w-[18px] h-[18px] transition-colors", isLiked && "fill-current")} />
                 <span className="text-[13px] transition-colors">{likeCount || ''}</span>
            </Button>

            {/* Share */}
             <Button variant="ghost" size="sm" className="group flex items-center gap-1 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-full px-2 h-8 cursor-pointer" onClick={handleShare}>
                <Share className="w-[18px] h-[18px] group-hover:text-primary transition-colors" />
            </Button>
        </div>
        <ReplyModal 
            tweet={displayTweet} 
            isOpen={showReplyModal} 
            onClose={() => setShowReplyModal(false)} 
        />
      </div>
    </article>
  );
}
