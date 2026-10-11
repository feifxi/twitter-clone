package com.fei.twitter.user;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import java.time.Instant;

/**
 * An account. Follower/following counts are maintained by SQL in {@link FollowRepository}
 * (atomically, together with the follow row), so they're read-only here.
 */
@Entity
@Table(name = "users")
public class User {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false)
    private String username;

    @Column(nullable = false, updatable = false)
    private String email;

    private String displayName;
    private String bio;
    private String location;
    private String website;
    private String avatarUrl;
    private String bannerUrl;

    @Column(nullable = false, updatable = false)
    private String role = "USER";

    @Column(nullable = false, updatable = false)
    private String provider;

    @Column(insertable = false, updatable = false)
    private int followersCount;

    @Column(insertable = false, updatable = false)
    private int followingCount;

    private Instant onboardedAt;

    @Column(insertable = false, updatable = false)
    private Instant createdAt;

    private Instant updatedAt;

    protected User() {}

    public User(String username, String email, String displayName, String avatarUrl, String provider) {
        this.username = username;
        this.email = email;
        this.displayName = displayName;
        this.avatarUrl = avatarUrl;
        this.provider = provider;
        this.updatedAt = Instant.now();
    }

    /** Sets the chosen handle and name; the first time, this completes onboarding. */
    public void completeOnboarding(String username, String displayName, Instant now) {
        this.username = username;
        this.displayName = displayName;
        if (onboardedAt == null) {
            onboardedAt = now;
        }
        updatedAt = now;
    }

    public void updateProfile(String displayName, String bio, String location, String website, String avatarUrl, String bannerUrl, Instant now) {
        this.displayName = displayName;
        this.bio = bio;
        this.location = location;
        this.website = website;
        this.avatarUrl = avatarUrl;
        this.bannerUrl = bannerUrl;
        this.updatedAt = now;
    }

    public Long getId() { return id; }
    public String getUsername() { return username; }
    public String getEmail() { return email; }
    public String getDisplayName() { return displayName; }
    public String getBio() { return bio; }
    public String getLocation() { return location; }
    public String getWebsite() { return website; }
    public String getAvatarUrl() { return avatarUrl; }
    public String getBannerUrl() { return bannerUrl; }
    public int getFollowersCount() { return followersCount; }
    public int getFollowingCount() { return followingCount; }
    public boolean isOnboarded() { return onboardedAt != null; }
    public Instant getCreatedAt() { return createdAt; }
}
