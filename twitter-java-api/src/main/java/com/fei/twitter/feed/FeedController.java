package com.fei.twitter.feed;

import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.shared.web.PageResponse;
import com.fei.twitter.tweet.TweetResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Feeds")
@RestController
@RequestMapping("/api/v1")
class FeedController {

    private final FeedService feeds;

    FeedController(FeedService feeds) {
        this.feeds = feeds;
    }

    @Operation(summary = "The ranked \"For you\" feed")
    @GetMapping("/feeds/global")
    PageResponse<TweetResponse> global(PageRequest page, Viewer viewer) {
        return PageResponse.of(feeds.forYou(page, viewer.id()), TweetResponse::of);
    }

    @Operation(summary = "Tweets and retweets by accounts you follow, newest first")
    @GetMapping("/feeds/following")
    PageResponse<TweetResponse> following(PageRequest page, Viewer viewer) {
        return PageResponse.of(feeds.following(viewer.requireId(), page), TweetResponse::of);
    }

    @Operation(summary = "Profile tab: top-level tweets and retweets")
    @GetMapping("/users/{id}/posts")
    PageResponse<TweetResponse> posts(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(feeds.posts(id, page, viewer.id()), TweetResponse::of);
    }

    @Operation(summary = "Profile tab: replies")
    @GetMapping("/users/{id}/replies")
    PageResponse<TweetResponse> replies(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(feeds.replies(id, page, viewer.id()), TweetResponse::of);
    }

    @Operation(summary = "Profile tab: tweets with media")
    @GetMapping("/users/{id}/media")
    PageResponse<TweetResponse> media(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(feeds.media(id, page, viewer.id()), TweetResponse::of);
    }

    @Operation(summary = "Profile tab: your likes (private)")
    @GetMapping("/users/{id}/likes")
    PageResponse<TweetResponse> likes(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(feeds.likes(id, page, viewer.id()), TweetResponse::of);
    }
}
