package com.fei.twitter.search;

import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.shared.web.PageResponse;
import com.fei.twitter.tweet.TweetResponse;
import com.fei.twitter.user.UserResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Search")
@RestController
@RequestMapping("/api/v1/search")
class SearchController {

    private final SearchService search;

    SearchController(SearchService search) {
        this.search = search;
    }

    @Operation(summary = "Users by handle or name, most-followed first")
    @GetMapping("/users")
    PageResponse<UserResponse> users(@RequestParam(defaultValue = "") String q, PageRequest page, Viewer viewer) {
        return PageResponse.of(search.users(required(q), page, viewer.id()), UserResponse::of);
    }

    @Operation(summary = "Tweets: \"#tag\" by hashtag (newest first), otherwise full-text by relevance")
    @GetMapping("/tweets")
    PageResponse<TweetResponse> tweets(@RequestParam(defaultValue = "") String q, PageRequest page, Viewer viewer) {
        return PageResponse.of(search.tweets(required(q), page, viewer.id()), TweetResponse::of);
    }

    @Operation(summary = "Hashtags by prefix, most used first")
    @GetMapping("/hashtags")
    List<HashtagResponse> hashtags(@RequestParam(defaultValue = "") String q, @RequestParam(defaultValue = "5") String limit) {
        return search.hashtags(q, Limits.parse(limit, 5));
    }

    private static String required(String q) {
        if (q.isBlank()) {
            throw ApiException.badRequest("search query is required");
        }
        return q.strip();
    }
}
