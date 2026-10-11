package com.fei.twitter.discovery;

import com.fei.twitter.search.HashtagQueries;
import com.fei.twitter.search.HashtagResponse;
import com.fei.twitter.search.Limits;
import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.web.Cursor;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.shared.web.PageResponse;
import com.fei.twitter.user.UserQueries;
import com.fei.twitter.user.UserResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import java.util.List;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/** What to read and who to follow. Small enough not to need a separate service. */
@Tag(name = "Discovery")
@RestController
@RequestMapping("/api/v1/discovery")
@Transactional(readOnly = true)
class DiscoveryController {

    private final HashtagQueries hashtags;
    private final JdbcClient jdbc;

    DiscoveryController(HashtagQueries hashtags, JdbcClient jdbc) {
        this.hashtags = hashtags;
        this.jdbc = jdbc;
    }

    @Operation(summary = "Hashtags trending in the last 24 hours (all-time when it's quiet)")
    @GetMapping("/trending")
    List<HashtagResponse> trending(@RequestParam(defaultValue = "10") String limit) {
        int n = Limits.parse(limit, 10);
        List<HashtagResponse> recent = hashtags.trendingLastDay(n);
        return recent.isEmpty() ? hashtags.topAllTime(n) : recent;
    }

    @Operation(summary = "Most-followed onboarded accounts you don't follow yet")
    @GetMapping("/users")
    PageResponse<UserResponse> suggestedUsers(PageRequest page, Viewer viewer) {
        var rows = jdbc.sql("SELECT " + UserQueries.COLUMNS + ", false AS following FROM users u"
                        + " WHERE u.onboarded_at IS NOT NULL"
                        + "   AND (CAST(:viewer AS bigint) IS NULL OR u.id <> :viewer)"
                        + "   AND NOT EXISTS (SELECT 1 FROM follows f WHERE f.following_id = u.id AND f.follower_id = :viewer)"
                        + " ORDER BY u.followers_count DESC, u.id DESC LIMIT :limit OFFSET :offset")
                .param("viewer", viewer.id())
                .param("limit", page.fetchLimit())
                .param("offset", page.offset())
                .query((rs, n) -> UserQueries.view(rs, ""))
                .list();
        return PageResponse.of(Page.fromRows(rows, page, u -> Cursor.ofOffset(page.offset() + page.limit())), UserResponse::of);
    }
}
