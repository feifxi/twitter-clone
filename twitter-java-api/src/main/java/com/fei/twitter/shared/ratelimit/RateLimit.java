package com.fei.twitter.shared.ratelimit;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;

/**
 * Applies an extra rate limit to a handler, on top of the API-wide one. Limits are per
 * route and client IP; with Redis they're shared across instances.
 */
@Target(ElementType.METHOD)
@Retention(RetentionPolicy.RUNTIME)
public @interface RateLimit {

    Policy value();

    enum Policy {
        /** Every /api/v1 request: 20/s, bursts of 60. */
        API(20, 60),
        /** Mutations: 2/s, bursts of 5. */
        WRITE(2, 5),
        /** Sign-in endpoints. */
        AUTH(2, 5),
        /** Calls that cost money (Gemini): one per 5 s, bursts of 5. */
        AI(0.2, 5);

        public final double perSecond;
        public final int burst;

        Policy(double perSecond, int burst) {
            this.perSecond = perSecond;
            this.burst = burst;
        }
    }
}
