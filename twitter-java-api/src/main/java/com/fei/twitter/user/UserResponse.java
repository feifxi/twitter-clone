package com.fei.twitter.user;

import java.time.Instant;

/** A public profile. Private fields (email) only appear in {@link CurrentUserResponse}. */
public record UserResponse(
        long id,
        String username,
        String displayName,
        String bio,
        String location,
        String website,
        String avatarUrl,
        String bannerUrl,
        boolean isFollowing,
        int followersCount,
        int followingCount,
        Instant createdAt) {

    public static UserResponse of(UserView u) {
        return new UserResponse(u.id(), u.username(), u.displayName(), u.bio(), u.location(), u.website(),
                u.avatarUrl(), u.bannerUrl(), u.following(), u.followersCount(), u.followingCount(), u.createdAt());
    }
}
