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
}

export interface RetweetResponse {
    id: number;
    retweeter: UserResponse;
    createdAt: string;
}

/**
 * One entry of a timeline (feeds and profile tabs). `tweet` is always the original;
 * for kind "retweet", `retweet` says who put it in this timeline and when.
 */
export interface FeedItemResponse {
    kind: 'tweet' | 'retweet';
    tweet: TweetResponse;
    retweet: RetweetResponse | null;
}
