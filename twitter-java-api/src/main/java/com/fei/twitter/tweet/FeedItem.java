package com.fei.twitter.tweet;

import com.fei.twitter.user.UserView;
import java.time.Instant;

/**
 * One entry of a timeline: a tweet, plus the retweet that put it there when it appears
 * because someone retweeted it.
 */
public record FeedItem(TweetView tweet, Retweet retweet) {

    public record Retweet(long id, UserView by, Instant createdAt) {}
}
