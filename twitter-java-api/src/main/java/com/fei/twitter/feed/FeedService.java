package com.fei.twitter.feed;

import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.tweet.TweetFeeds;
import com.fei.twitter.tweet.FeedItem;
import com.fei.twitter.user.UserService;
import java.time.Clock;
import java.time.Instant;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/** The home timelines and the profile tabs; all of them return feed items. */
@Service
@Transactional(readOnly = true)
public class FeedService {

    private final TweetFeeds feeds;
    private final UserService users;
    private final Clock clock;

    FeedService(TweetFeeds feeds, UserService users, Clock clock) {
        this.feeds = feeds;
        this.users = users;
        this.clock = clock;
    }

    /**
     * The ranked "For you" feed. The first page fixes an anchor time; later pages score
     * against that same anchor so the ranking doesn't drift while scrolling.
     */
    public Page<FeedItem> forYou(PageRequest page, Long viewerId) {
        Instant anchor = page.after() != null ? page.after().anchor() : clock.instant();
        return feeds.forYou(anchor, page, viewerId, r -> Cursor.ofScore(anchor, r.sortScore(), r.id()));
    }

    /** Tweets and retweets by accounts the user follows, newest first. */
    public Page<FeedItem> following(long userId, PageRequest page) {
        return feeds.timeline(false, userId, page, userId);
    }

    public Page<FeedItem> posts(long userId, PageRequest page, Long viewerId) {
        users.get(userId, null);
        return feeds.timeline(true, userId, page, viewerId);
    }

    public Page<FeedItem> replies(long userId, PageRequest page, Long viewerId) {
        users.get(userId, null);
        return feeds.userReplies(userId, page, viewerId);
    }

    public Page<FeedItem> media(long userId, PageRequest page, Long viewerId) {
        users.get(userId, null);
        return feeds.userMedia(userId, page, viewerId);
    }

    /** Likes are private, as on X: only the user themselves may list them. */
    public Page<FeedItem> likes(long userId, PageRequest page, Long viewerId) {
        if (viewerId == null || viewerId != userId) {
            throw ApiException.forbidden("likes are only visible to their owner");
        }
        return feeds.userLikes(userId, page, viewerId, r -> Cursor.ofTime(r.sortTime(), r.id()));
    }
}
