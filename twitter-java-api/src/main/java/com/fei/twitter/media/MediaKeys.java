package com.fei.twitter.media;

import com.fei.twitter.shared.error.ApiException;
import java.util.Map;
import java.util.Set;
import java.util.UUID;

/**
 * Object keys look like {@code <folder>/<userId>/<uuid>.<ext>}. Clients send back the key
 * they uploaded to; checking the folder and owner stops them attaching someone else's file.
 */
public final class MediaKeys {

    public static final String TWEETS = "tweets";
    public static final String AVATARS = "avatars";
    public static final String BANNERS = "banners";

    static final Map<String, String> EXTENSIONS = Map.of(
            "image/jpeg", ".jpg", "image/png", ".png", "image/gif", ".gif", "image/webp", ".webp",
            "video/mp4", ".mp4", "video/webm", ".webm");
    private static final Set<String> VIDEO_EXTENSIONS = Set.of(".mp4", ".webm");

    private MediaKeys() {}

    static String newKey(String folder, long userId, String extension) {
        return folder + "/" + userId + "/" + UUID.randomUUID() + extension;
    }

    public static void validateOwned(String key, String folder, long userId) {
        String prefix = folder + "/" + userId + "/";
        String name = key.startsWith(prefix) ? key.substring(prefix.length()) : "";
        if (name.isEmpty() || name.contains("/") || name.contains("\\") || !EXTENSIONS.containsValue(extension(name))) {
            throw ApiException.badRequest("invalid media key");
        }
    }

    /** IMAGE or VIDEO, from the key's extension. */
    public static String typeOf(String key) {
        return VIDEO_EXTENSIONS.contains(extension(key)) ? "VIDEO" : "IMAGE";
    }

    private static String extension(String name) {
        int dot = name.lastIndexOf('.');
        return dot < 0 ? "" : name.substring(dot);
    }
}
