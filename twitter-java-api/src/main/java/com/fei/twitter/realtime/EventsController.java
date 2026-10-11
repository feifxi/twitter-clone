package com.fei.twitter.realtime;

import com.fei.twitter.security.Viewer;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

@Tag(name = "Realtime")
@RestController
class EventsController {

    private final RealtimeHub hub;

    EventsController(RealtimeHub hub) {
        this.hub = hub;
    }

    @Operation(summary = "One Server-Sent Events stream per tab: 'connected', then 'notification' and 'dm' events")
    @GetMapping(path = "/api/v1/events", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    SseEmitter events(Viewer viewer, HttpServletResponse response) {
        response.setHeader("Cache-Control", "no-cache");
        response.setHeader("X-Accel-Buffering", "no");
        return hub.subscribe(viewer.requireId());
    }
}
