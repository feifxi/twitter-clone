package com.fei.twitter.tweet;

import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import java.time.Instant;
import java.util.function.Function;
import org.springframework.stereotype.Component;

/** Paged, hydrated tweet lists for the feed and search modules. */
@Component
public class TweetFeeds {

    private final TweetQueries queries;
    private final TweetHydrator hydrator;

    TweetFeeds(TweetQueries queries, TweetHydrator hydrator) {
        this.queries = queries;
        this.hydrator = hydrator;
    }

    public Page<FeedItem> forYou(Instant anchor, PageRequest page, Long viewerId, Function<TweetRow, Cursor> cursorOf) {
        return hydrator.itemPage(queries.forYou(anchor, page, viewerId), page, viewerId, cursorOf);
    }

    /** {@code profile}: one user's tweets and retweets; otherwise those of the accounts {@code owner} follows. */
    public Page<FeedItem> timeline(boolean profile, long owner, PageRequest page, Long viewerId) {
        return hydrator.itemPage(queries.timeline(profile, owner, page, viewerId), page, viewerId, TweetRow::timelineCursor);
    }

    public Page<FeedItem> userReplies(long userId, PageRequest page, Long viewerId) {
        return hydrator.itemPage(queries.userReplies(userId, page, viewerId), page, viewerId, TweetRow::createdCursor);
    }

    public Page<FeedItem> userMedia(long userId, PageRequest page, Long viewerId) {
        return hydrator.itemPage(queries.userMedia(userId, page, viewerId), page, viewerId, TweetRow::createdCursor);
    }

    public Page<FeedItem> userLikes(long userId, PageRequest page, Long viewerId, Function<TweetRow, Cursor> cursorOf) {
        return hydrator.itemPage(queries.userLikes(userId, page, viewerId), page, viewerId, cursorOf);
    }

    public Page<TweetView> searchFullText(String tsQuery, PageRequest page, Long viewerId) {
        var rows = queries.searchFullText(tsQuery, page, viewerId);
        return hydrator.page(rows, page, viewerId, r -> Cursor.ofOffset(page.offset() + page.limit()));
    }

    public Page<TweetView> searchByHashtag(String tag, PageRequest page, Long viewerId) {
        return hydrator.page(queries.searchByHashtag(tag, page, viewerId), page, viewerId, TweetRow::createdCursor);
    }
}
