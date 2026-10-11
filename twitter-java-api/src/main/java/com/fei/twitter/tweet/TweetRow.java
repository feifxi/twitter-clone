package com.fei.twitter.tweet;

import com.fei.twitter.shared.web.Cursor;
import java.time.Instant;

/**
 * A tweet row with the viewer's flags and, for timelines, the retweet that put it there.
 * Every tweet-listing query returns this shape (see {@link TweetQueries#SELECT}).
 */
public record TweetRow(
        long id,
        long userId,
        String content,
        String mediaType,
        String mediaUrl,
        Long parentId,
        int replyCount,
        int likeCount,
        int retweetCount,
        Instant createdAt,
        boolean liked,
        boolean retweeted,
        Long retweetId,
        Long retweetedById,
        Instant retweetedAt,
        double sortScore,
        Instant sortTime) {

    /** Mixed tweet/retweet timelines order by (time, kind, id): kind 0 = tweet, 1 = retweet. */
    public Cursor timelineCursor() {
        return retweetId != null ? Cursor.ofTimeline(retweetedAt, 1, retweetId) : Cursor.ofTime(createdAt, id);
    }

    public Cursor createdCursor() {
        return Cursor.ofTime(createdAt, id);
    }
}
