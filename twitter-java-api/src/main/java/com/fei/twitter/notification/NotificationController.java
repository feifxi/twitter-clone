package com.fei.twitter.notification;

import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.ratelimit.RateLimit;
import com.fei.twitter.shared.web.PageRequest;
import com.fei.twitter.shared.web.PageResponse;
import com.fei.twitter.shared.web.SuccessResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Notifications")
@RestController
@RequestMapping("/api/v1/notifications")
class NotificationController {

    private final NotificationService notifications;

    NotificationController(NotificationService notifications) {
        this.notifications = notifications;
    }

    record UnreadCountResponse(long count) {}

    @Operation(summary = "Your notifications, newest first")
    @GetMapping
    PageResponse<NotificationResponse> list(PageRequest page, Viewer viewer) {
        return PageResponse.of(notifications.list(viewer.requireId(), page), n -> n);
    }

    @Operation(summary = "How many are unread")
    @GetMapping("/unread-count")
    UnreadCountResponse unreadCount(Viewer viewer) {
        return new UnreadCountResponse(notifications.unreadCount(viewer.requireId()));
    }

    @Operation(summary = "Mark everything read")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/mark-read")
    SuccessResponse markRead(Viewer viewer) {
        notifications.markAllRead(viewer.requireId());
        return SuccessResponse.OK;
    }
}
