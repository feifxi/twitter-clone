import type { UserResponse } from './user';

export interface TweetRequest {
    content?: string;
    parentId?: number | null;
}

export interface TweetResponse {
    id: number;
    content: string | null;
    mediaType: string | null;
    mediaUrl: string | null;
    user: UserResponse;
    replyCount: number;
    likeCount: number;
    retweetCount: number;
    isLiked: boolean;
    isRetweeted: boolean;
    replyToTweetId?: number | null;
    replyToUsername?: string | null;
    createdAt: string;
    /** Set when the tweet is in a timeline because this user retweeted it. */
    retweetedBy: UserResponse | null;
    retweetedAt: string | null;
}
