package com.fei.twitter.user;

import java.time.Instant;

/** A user as seen by a viewer: profile fields plus whether the viewer follows them. */
public record UserView(
        long id,
        String username,
        String displayName,
        String bio,
        String location,
        String website,
        String avatarUrl,
        String bannerUrl,
        int followersCount,
        int followingCount,
        Instant createdAt,
        boolean following) {

    public static UserView of(User u, boolean following) {
        return new UserView(u.getId(), u.getUsername(), u.getDisplayName(), u.getBio(), u.getLocation(), u.getWebsite(),
                u.getAvatarUrl(), u.getBannerUrl(), u.getFollowersCount(), u.getFollowingCount(), u.getCreatedAt(), following);
    }
}
