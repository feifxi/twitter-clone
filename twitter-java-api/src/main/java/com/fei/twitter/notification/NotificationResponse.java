package com.fei.twitter.notification;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fei.twitter.user.UserResponse;
import java.time.Instant;

/** For replies, the original* fields describe the tweet that was replied to. */
public record NotificationResponse(
        long id,
        UserResponse actor,
        Long tweetId,
        String tweetContent,
        String tweetMediaUrl,
        @JsonInclude(JsonInclude.Include.NON_NULL) Long originalTweetId,
        @JsonInclude(JsonInclude.Include.NON_NULL) String originalTweetContent,
        @JsonInclude(JsonInclude.Include.NON_NULL) String originalTweetMediaUrl,
        String type,
        boolean isRead,
        Instant createdAt) {}
