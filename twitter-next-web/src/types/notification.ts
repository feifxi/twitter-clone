import type { UserResponse } from './user';

export type NotificationType = 'LIKE' | 'RETWEET' | 'REPLY' | 'FOLLOW';

/** What a notification shows of a post. */
export interface TweetPreviewResponse {
    id: number;
    content: string | null;
    mediaType: string | null;
    mediaUrl: string | null;
}

export interface NotificationResponse {
    id: number;
    type: NotificationType;
    actor: UserResponse;
    /** The post it's about (null for follows); for a reply, the reply itself. */
    tweet: TweetPreviewResponse | null;
    /** For a reply: the post that was replied to. */
    originalTweet: TweetPreviewResponse | null;
    isRead: boolean;
    createdAt: string;
}
