package com.fei.twitter.tweet;

import com.fei.twitter.user.UserResponse;
import java.time.Instant;

/**
 * One entry of a timeline (feeds and profile tabs). {@code kind} is "retweet" when the
 * tweet is there because someone retweeted it; {@code retweet} is then set (and null
 * otherwise). The tweet is always the original, in the same shape as GET /tweets/{id}.
 */
public record FeedItemResponse(String kind, TweetResponse tweet, RetweetResponse retweet) {

    public record RetweetResponse(long id, UserResponse retweeter, Instant createdAt) {}

    public static FeedItemResponse of(FeedItem item) {
        if (item.retweet() == null) {
            return new FeedItemResponse("tweet", TweetResponse.of(item.tweet()), null);
        }
        var retweet = item.retweet();
        return new FeedItemResponse("retweet", TweetResponse.of(item.tweet()),
                new RetweetResponse(retweet.id(), UserResponse.of(retweet.by()), retweet.createdAt()));
    }
}
