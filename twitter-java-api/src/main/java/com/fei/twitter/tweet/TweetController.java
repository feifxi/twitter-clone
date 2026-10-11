package com.fei.twitter.tweet;

import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.ratelimit.RateLimit;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.shared.web.PageResponse;
import com.fei.twitter.shared.web.SuccessResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.Size;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Tweets")
@RestController
@RequestMapping("/api/v1/tweets")
class TweetController {

    private final TweetService tweets;

    TweetController(TweetService tweets) {
        this.tweets = tweets;
    }

    /** Text, media (an uploaded object key), or both. */
    record CreateTweetRequest(
            @Size(max = 280, message = "must be at most 280 characters") String content,
            @Min(value = 1, message = "must be at least 1") Long parentId,
            @Size(max = 200, message = "must be at most 200 characters") String mediaKey) {}

    @Operation(summary = "A tweet")
    @GetMapping("/{id}")
    TweetResponse get(@PathVariable long id, Viewer viewer) {
        return TweetResponse.of(tweets.get(id, viewer.id()));
    }

    @Operation(summary = "A tweet's direct replies, oldest first")
    @GetMapping("/{id}/replies")
    PageResponse<TweetResponse> replies(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(tweets.replies(id, page, viewer.id()), TweetResponse::of);
    }

    @Operation(summary = "Post a tweet or a reply")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping
    @ResponseStatus(HttpStatus.CREATED)
    TweetResponse create(@Valid @RequestBody CreateTweetRequest r, Viewer viewer) {
        return TweetResponse.of(tweets.create(viewer.requireId(), r.content(), r.parentId(), r.mediaKey()));
    }

    @Operation(summary = "Delete your tweet (and its replies)")
    @RateLimit(RateLimit.Policy.WRITE)
    @DeleteMapping("/{id}")
    SuccessResponse delete(@PathVariable long id, Viewer viewer) {
        tweets.delete(viewer.requireId(), id);
        return SuccessResponse.OK;
    }

    @Operation(summary = "Like a tweet")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/{id}/like")
    SuccessResponse like(@PathVariable long id, Viewer viewer) {
        tweets.like(viewer.requireId(), id);
        return SuccessResponse.OK;
    }

    @Operation(summary = "Remove your like")
    @RateLimit(RateLimit.Policy.WRITE)
    @DeleteMapping("/{id}/like")
    SuccessResponse unlike(@PathVariable long id, Viewer viewer) {
        tweets.unlike(viewer.requireId(), id);
        return SuccessResponse.OK;
    }

    @Operation(summary = "Retweet (idempotent)")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/{id}/retweet")
    TweetResponse retweet(@PathVariable long id, Viewer viewer) {
        return TweetResponse.of(tweets.retweet(viewer.requireId(), id));
    }

    @Operation(summary = "Undo your retweet (idempotent)")
    @RateLimit(RateLimit.Policy.WRITE)
    @DeleteMapping("/{id}/retweet")
    SuccessResponse undoRetweet(@PathVariable long id, Viewer viewer) {
        tweets.undoRetweet(viewer.requireId(), id);
        return SuccessResponse.OK;
    }
}
