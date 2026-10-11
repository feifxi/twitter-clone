package com.fei.twitter.tweet;

import com.fei.twitter.user.UserView;
import java.time.Instant;

/** A tweet as a viewer sees it: author loaded, viewer flags set. */
public record TweetView(
        long id,
        String content,
        String mediaType,
        String mediaUrl,
        UserView author,
        int replyCount,
        int likeCount,
        int retweetCount,
        boolean liked,
        boolean retweeted,
        Long parentId,
        String parentUsername,
        Instant createdAt) {}
