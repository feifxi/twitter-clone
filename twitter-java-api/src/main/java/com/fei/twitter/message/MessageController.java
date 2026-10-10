package com.fei.twitter.message;

import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.ratelimit.RateLimit;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.shared.web.PageResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Messages")
@RestController
@RequestMapping("/api/v1/messages")
class MessageController {

    private final MessageService messages;

    MessageController(MessageService messages) {
        this.messages = messages;
    }

    /** Length is checked in characters by the service (2000). */
    record SendMessageRequest(@NotBlank(message = "this field is required") String content) {}

    @Operation(summary = "Your conversations, most recently active first")
    @GetMapping("/conversations")
    PageResponse<ConversationResponse> conversations(PageRequest page, Viewer viewer) {
        return PageResponse.of(messages.conversations(viewer.requireId(), page), c -> c);
    }

    @Operation(summary = "A conversation's messages: newest page first, each page oldest-first")
    @GetMapping("/conversations/{id}/messages")
    PageResponse<MessageResponse> messages(@PathVariable long id, PageRequest page, Viewer viewer) {
        return PageResponse.of(messages.messages(viewer.requireId(), id, page), m -> m);
    }

    @Operation(summary = "Send to a conversation you're in")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/conversations/{id}/messages")
    @ResponseStatus(HttpStatus.CREATED)
    MessageResponse sendToConversation(@PathVariable long id, @Valid @RequestBody SendMessageRequest r, Viewer viewer) {
        return messages.sendToConversation(viewer.requireId(), id, r.content());
    }

    @Operation(summary = "Send to a user (creates the conversation on first contact)")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/users/{id}/messages")
    @ResponseStatus(HttpStatus.CREATED)
    MessageResponse sendToUser(@PathVariable long id, @Valid @RequestBody SendMessageRequest r, Viewer viewer) {
        return messages.sendToUser(viewer.requireId(), id, r.content());
    }
}
