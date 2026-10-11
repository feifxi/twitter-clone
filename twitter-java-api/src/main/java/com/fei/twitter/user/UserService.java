package com.fei.twitter.user;

import com.fei.twitter.media.MediaKeys;
import com.fei.twitter.media.MediaStorage;
import com.fei.twitter.notification.NotificationType;
import com.fei.twitter.notification.Notifications;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.web.Page;
import com.fei.twitter.shared.web.PageRequest;
import java.time.Clock;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.regex.Pattern;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/** Profiles, onboarding and the follow graph. */
@Service
public class UserService {

    private static final Pattern USERNAME = Pattern.compile("^[A-Za-z0-9_]{3,20}$");

    /** Profiles live at /{username} in the web app; these would be shadowed by its own pages (or look official). */
    private static final Set<String> RESERVED = Set.of(
            "admin", "api", "brainrot", "chanombot", "connect_people", "explore", "home", "login", "logout",
            "messages", "notifications", "onboarding", "premium", "search", "settings", "support", "tweet");

    private final UserRepository users;
    private final UserQueries queries;
    private final FollowRepository follows;
    private final Notifications notifications;
    private final MediaStorage storage;
    private final Clock clock;

    UserService(UserRepository users, UserQueries queries, FollowRepository follows, Notifications notifications,
                MediaStorage storage, Clock clock) {
        this.users = users;
        this.queries = queries;
        this.follows = follows;
        this.notifications = notifications;
        this.storage = storage;
        this.clock = clock;
    }

    /** Profile edits: null leaves a field unchanged, "" clears it. Image keys come from presigned uploads. */
    public record ProfileUpdate(String displayName, String bio, String location, String website, String avatarKey, String bannerKey) {}

    @Transactional(readOnly = true)
    public UserView get(long userId, Long viewerId) {
        UserView user = queries.byIds(List.of(userId), viewerId).get(userId);
        if (user == null) {
            throw ApiException.notFound("user");
        }
        return user;
    }

    @Transactional(readOnly = true)
    public UserView getByUsername(String username, Long viewerId) {
        User user = users.findByUsernameIgnoreCase(username).orElseThrow(() -> ApiException.notFound("user"));
        return get(user.getId(), viewerId);
    }

    @Transactional
    public User completeOnboarding(long userId, String username, String displayName) {
        if (!USERNAME.matcher(username).matches()) {
            throw ApiException.badRequest("username must be 3-20 letters, numbers or underscores");
        }
        if (RESERVED.contains(username.toLowerCase(Locale.ROOT))) {
            throw ApiException.conflict("that username isn't available");
        }
        String name = displayName.strip();
        if (name.isEmpty()) {
            throw ApiException.badRequest("display name is required");
        }
        User user = users.findById(userId).orElseThrow(() -> ApiException.notFound("user"));
        users.findByUsernameIgnoreCase(username)
                .filter(other -> !other.getId().equals(userId))
                .ifPresent(other -> {
                    throw ApiException.conflict("username is already taken");
                });
        user.completeOnboarding(username, name, clock.instant());
        try {
            return users.saveAndFlush(user);
        } catch (DataIntegrityViolationException e) {
            throw ApiException.conflict("username is already taken"); // lost a race for the same handle
        }
    }

    @Transactional
    public User updateProfile(long userId, ProfileUpdate update) {
        User user = users.findById(userId).orElseThrow(() -> ApiException.notFound("user"));
        String avatar = resolveImage(user.getAvatarUrl(), update.avatarKey(), MediaKeys.AVATARS, userId);
        String banner = resolveImage(user.getBannerUrl(), update.bannerKey(), MediaKeys.BANNERS, userId);
        String oldAvatar = user.getAvatarUrl();
        String oldBanner = user.getBannerUrl();

        user.updateProfile(
                merge(user.getDisplayName(), update.displayName()),
                merge(user.getBio(), update.bio()),
                merge(user.getLocation(), update.location()),
                merge(user.getWebsite(), update.website()),
                avatar, banner, clock.instant());
        User saved = users.saveAndFlush(user);

        // Replaced images are deleted from our bucket (external URLs, e.g. Google avatars, are left alone).
        if (update.avatarKey() != null) {
            storage.keyFromPublicUrl(oldAvatar).ifPresent(storage::delete);
        }
        if (update.bannerKey() != null) {
            storage.keyFromPublicUrl(oldBanner).ifPresent(storage::delete);
        }
        return saved;
    }

    /** @return whether a new follow was created (false if it already existed) */
    @Transactional
    public boolean follow(long followerId, long targetId) {
        if (followerId == targetId) {
            throw ApiException.badRequest("cannot follow yourself");
        }
        get(targetId, null);
        boolean created = follows.follow(followerId, targetId);
        if (created) {
            notifications.create(targetId, followerId, null, NotificationType.FOLLOW);
        }
        return created;
    }

    @Transactional
    public void unfollow(long followerId, long targetId) {
        get(targetId, null);
        follows.unfollow(followerId, targetId);
    }

    @Transactional(readOnly = true)
    public Page<UserView> followers(long userId, PageRequest page, Long viewerId) {
        get(userId, viewerId);
        return follows.followers(userId, page, viewerId);
    }

    @Transactional(readOnly = true)
    public Page<UserView> following(long userId, PageRequest page, Long viewerId) {
        get(userId, viewerId);
        return follows.following(userId, page, viewerId);
    }

    private String resolveImage(String current, String key, String folder, long userId) {
        if (key == null) {
            return current;
        }
        if (key.isEmpty()) {
            return null;
        }
        MediaKeys.validateOwned(key, folder, userId);
        return storage.publicUrl(key);
    }

    private static String merge(String current, String update) {
        if (update == null) {
            return current;
        }
        return update.isEmpty() ? null : update;
    }
}
