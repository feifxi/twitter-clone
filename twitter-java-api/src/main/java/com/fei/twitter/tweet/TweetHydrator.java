package com.fei.twitter.tweet;

import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.user.UserQueries;
import com.fei.twitter.user.UserView;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.function.Function;
import java.util.stream.Collectors;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Component;

/**
 * Turns {@link TweetRow}s into {@link TweetView}s (or, for timelines, {@link FeedItem}s
 * with the retweet that put a tweet there): authors, retweeters and the handles of
 * replied-to authors are loaded in two batch queries per page.
 */
@Component
public class TweetHydrator {

    private final UserQueries users;
    private final JdbcClient jdbc;

    TweetHydrator(UserQueries users, JdbcClient jdbc) {
        this.users = users;
        this.jdbc = jdbc;
    }

    public List<TweetView> hydrate(List<TweetRow> rows, Long viewerId) {
        return hydrateItems(rows, viewerId).stream().map(FeedItem::tweet).toList();
    }

    public List<FeedItem> hydrateItems(List<TweetRow> rows, Long viewerId) {
        if (rows.isEmpty()) {
            return List.of();
        }
        Set<Long> parentIds = rows.stream().map(TweetRow::parentId).filter(Objects::nonNull).collect(Collectors.toSet());
        Map<Long, Long> parentAuthors = parentIds.isEmpty() ? Map.of() : jdbc.sql("SELECT id, user_id FROM tweets WHERE id IN (:ids)")
                .param("ids", parentIds)
                .query((rs, n) -> Map.entry(rs.getLong("id"), rs.getLong("user_id")))
                .list().stream().collect(Collectors.toMap(Map.Entry::getKey, Map.Entry::getValue));

        Set<Long> userIds = new HashSet<>(parentAuthors.values());
        for (TweetRow r : rows) {
            userIds.add(r.userId());
            if (r.retweetedById() != null) {
                userIds.add(r.retweetedById());
            }
        }
        Map<Long, UserView> loaded = users.byIds(userIds, viewerId);

        return rows.stream().map(r -> {
            Long parentAuthor = r.parentId() == null ? null : parentAuthors.get(r.parentId());
            var tweet = new TweetView(r.id(), r.content(), r.mediaType(), r.mediaUrl(), loaded.get(r.userId()),
                    r.replyCount(), r.likeCount(), r.retweetCount(), r.liked(), r.retweeted(),
                    r.parentId(), parentAuthor == null ? null : loaded.get(parentAuthor).username(), r.createdAt());
            var retweet = r.retweetId() == null ? null
                    : new FeedItem.Retweet(r.retweetId(), loaded.get(r.retweetedById()), r.retweetedAt());
            return new FeedItem(tweet, retweet);
        }).toList();
    }

    /** Builds a page from rows fetched with {@link PageRequest#fetchLimit()}. */
    public Page<TweetView> page(List<TweetRow> rows, PageRequest request, Long viewerId, Function<TweetRow, Cursor> cursorOf) {
        Page<TweetRow> page = Page.fromRows(rows, request, cursorOf);
        return new Page<>(hydrate(page.items(), viewerId), page.next());
    }

    /** {@link #page} for timelines: the items keep the retweet that put a tweet there. */
    public Page<FeedItem> itemPage(List<TweetRow> rows, PageRequest request, Long viewerId, Function<TweetRow, Cursor> cursorOf) {
        Page<TweetRow> page = Page.fromRows(rows, request, cursorOf);
        return new Page<>(hydrateItems(page.items(), viewerId), page.next());
    }
}
