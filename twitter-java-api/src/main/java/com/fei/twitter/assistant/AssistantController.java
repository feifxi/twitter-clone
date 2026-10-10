package com.fei.twitter.assistant;

import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.ratelimit.RateLimit;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.servlet.http.HttpServletResponse;
import jakarta.validation.Valid;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Pattern;
import jakarta.validation.constraints.Size;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.codec.ServerSentEvent;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;
import reactor.core.publisher.Flux;

@Tag(name = "Assistant")
@RestController
class AssistantController {

    private static final Logger log = LoggerFactory.getLogger(AssistantController.class);

    private final AssistantService assistant;

    AssistantController(AssistantService assistant) {
        this.assistant = assistant;
    }

    record HistoryItem(
            @NotBlank(message = "this field is required")
            @Pattern(regexp = "user|model|assistant", message = "must be one of [user model assistant]") String role,
            @Size(max = 4000, message = "must be at most 4000 characters") String text) {}

    record AssistantRequest(
            @NotBlank(message = "this field is required") @Size(max = 4000, message = "must be at most 4000 characters") String query,
            @Size(max = 100, message = "must have at most 100 items") List<@Valid HistoryItem> history) {}

    /**
     * Streams the answer as Server-Sent Events: {@code data: {"text": "..."}} chunks, then
     * {@code event: done}, or {@code event: error} if the model fails mid-stream.
     */
    @Operation(summary = "Ask ChanomBot (streamed as Server-Sent Events)")
    @RateLimit(RateLimit.Policy.AI)
    @PostMapping(path = "/api/v1/assistant", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    Flux<ServerSentEvent<Object>> ask(@Valid @RequestBody AssistantRequest request, Viewer viewer, HttpServletResponse response) {
        viewer.requireId();
        List<AssistantService.Turn> history = request.history() == null ? List.of() : request.history().stream()
                .map(h -> new AssistantService.Turn(!h.role().equals("user"), h.text() == null ? "" : h.text()))
                .toList();
        response.setHeader("Cache-Control", "no-cache");
        response.setHeader("X-Accel-Buffering", "no");

        return assistant.chat(request.query(), history)
                .map(text -> ServerSentEvent.<Object>builder(Map.of("text", text)).build())
                .concatWith(Flux.just(ServerSentEvent.<Object>builder(Map.of()).event("done").build()))
                .onErrorResume(e -> {
                    log.error("assistant stream failed", e);
                    return Flux.just(ServerSentEvent.<Object>builder(Map.of("message", "the assistant is unavailable, please try again"))
                            .event("error").build());
                });
    }
}
