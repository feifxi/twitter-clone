package com.fei.twitter.message;

import com.fei.twitter.user.UserResponse;
import java.time.Instant;

public record ConversationResponse(long id, UserResponse peer, MessageResponse lastMessage, Instant updatedAt) {}
