package com.fei.twitter.tweet;

import com.fei.twitter.user.UserResponse;
import java.time.Instant;

public record TweetResponse(
        long id,
        String content,
        String mediaType,
        String mediaUrl,
        UserResponse user,
        int replyCount,
        int likeCount,
        int retweetCount,
        boolean isLiked,
        boolean isRetweeted,
        Long replyToTweetId,
        String replyToUsername,
        Instant createdAt,
        /** Set when the tweet is in a timeline because this user retweeted it. */
        UserResponse retweetedBy,
        Instant retweetedAt) {

    public static TweetResponse of(TweetView t) {
        return new TweetResponse(t.id(), t.content(), t.mediaType(), t.mediaUrl(), UserResponse.of(t.author()),
                t.replyCount(), t.likeCount(), t.retweetCount(), t.liked(), t.retweeted(),
                t.parentId(), t.parentUsername(), t.createdAt(),
                t.retweetedBy() == null ? null : UserResponse.of(t.retweetedBy()), t.retweetedAt());
    }
}
