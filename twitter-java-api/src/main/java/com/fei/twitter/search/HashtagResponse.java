package com.fei.twitter.search;

import java.time.Instant;

public record HashtagResponse(long id, String text, int usageCount, Instant lastUsedAt, Instant createdAt) {}
