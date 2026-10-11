package com.fei.twitter.search;

import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.tweet.TweetFeeds;
import com.fei.twitter.tweet.TweetView;
import com.fei.twitter.user.UserQueries;
import com.fei.twitter.user.UserView;
import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/** Finds users, tweets (full-text or by hashtag) and hashtags. */
@Service
@Transactional(readOnly = true)
public class SearchService {

    private final JdbcClient jdbc;
    private final TweetFeeds tweets;
    private final HashtagQueries hashtags;

    SearchService(JdbcClient jdbc, TweetFeeds tweets, HashtagQueries hashtags) {
        this.jdbc = jdbc;
        this.tweets = tweets;
        this.hashtags = hashtags;
    }

    /** Matches on handle or name, most-followed first; pages by offset. */
    public Page<UserView> users(String query, PageRequest page, Long viewerId) {
        List<UserView> rows = jdbc.sql("SELECT " + UserQueries.COLUMNS + ", " + UserQueries.FOLLOWING + " AS following FROM users u"
                        + " WHERE u.username ILIKE '%' || :q || '%' OR u.display_name ILIKE '%' || :q || '%'"
                        + " ORDER BY u.followers_count DESC, u.id DESC LIMIT :limit OFFSET :offset")
                .param("q", query.strip())
                .param("viewer", viewerId)
                .param("limit", page.fetchLimit())
                .param("offset", page.offset())
                .query((rs, n) -> UserQueries.view(rs, ""))
                .list();
        return Page.fromRows(rows, page, u -> Cursor.ofOffset(page.offset() + page.limit()));
    }

    /** "#tag" searches the hashtag (newest first); anything else is a relevance-ranked full-text search. */
    public Page<TweetView> tweets(String query, PageRequest page, Long viewerId) {
        String q = query.strip();
        if (q.startsWith("#")) {
            String tag = q.replaceFirst("^#+", "");
            return tag.isEmpty() ? empty() : tweets.searchByHashtag(tag, page, viewerId);
        }
        String tsQuery = toTsQuery(q);
        return tsQuery.isEmpty() ? empty() : tweets.searchFullText(tsQuery, page, viewerId);
    }

    public List<HashtagResponse> hashtags(String query, int limit) {
        String prefix = query.strip().replaceFirst("^#", "");
        return prefix.isEmpty() ? List.of() : hashtags.byPrefix(prefix, limit);
    }

    /** ANDs the alphanumeric words; everything else is dropped so input can't inject tsquery operators. */
    static String toTsQuery(String raw) {
        return Arrays.stream(raw.replaceAll("[^A-Za-z0-9]", " ").trim().split("\\s+"))
                .filter(w -> !w.isEmpty())
                .collect(Collectors.joining(" & "));
    }

    private static <T> Page<T> empty() {
        return new Page<>(List.of(), null);
    }
}
