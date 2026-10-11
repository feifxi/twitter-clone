package com.fei.twitter.message;

import com.fei.twitter.user.UserResponse;
import java.time.Instant;

public record MessageResponse(long id, long conversationId, UserResponse sender, String content, Instant createdAt) {}
