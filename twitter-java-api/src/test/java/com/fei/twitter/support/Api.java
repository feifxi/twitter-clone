package com.fei.twitter.support;

import com.fei.twitter.support.Sessions.TestUser;
import com.jayway.jsonpath.JsonPath;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

/** Small helpers for driving the API in tests. */
public final class Api {

    private Api() {}

    public static long tweet(MockMvcTester mvc, TestUser user, String content) {
        return post(mvc, user, "/api/v1/tweets", "{\"content\":\"" + content + "\"}");
    }

    public static long reply(MockMvcTester mvc, TestUser user, long parentId, String content) {
        return post(mvc, user, "/api/v1/tweets", "{\"content\":\"" + content + "\",\"parentId\":" + parentId + "}");
    }

    private static long post(MockMvcTester mvc, TestUser user, String path, String json) {
        var response = mvc.post().uri(path).cookie(user.cookie()).contentType(MediaType.APPLICATION_JSON).content(json).exchange().getResponse();
        if (response.getStatus() != 201) {
            throw new IllegalStateException(path + " → " + response.getStatus());
        }
        return ((Number) JsonPath.read(body(response), "$.id")).longValue();
    }

    public static void send(MockMvcTester mvc, TestUser user, String method, String path) {
        var request = switch (method) {
            case "POST" -> mvc.post().uri(path);
            case "DELETE" -> mvc.delete().uri(path);
            default -> throw new IllegalArgumentException(method);
        };
        int status = request.cookie(user.cookie()).exchange().getResponse().getStatus();
        if (status >= 300) {
            throw new IllegalStateException(method + " " + path + " → " + status);
        }
    }

    /** Follows every page of a list and returns all items. */
    public static List<Map<String, Object>> collect(MockMvcTester mvc, TestUser viewer, String path, int size) {
        List<Map<String, Object>> items = new ArrayList<>();
        String cursor = null;
        for (int i = 0; i < 100; i++) {
            String uri = path + (path.contains("?") ? "&" : "?") + "size=" + size + (cursor == null ? "" : "&cursor=" + cursor);
            var request = mvc.get().uri(java.net.URI.create(uri)); // already encoded: don't encode twice
            if (viewer != null) {
                request = request.cookie(viewer.cookie());
            }
            var response = request.exchange().getResponse();
            String json = body(response);
            if (response.getStatus() != 200) {
                throw new IllegalStateException(uri + " → " + response.getStatus() + " " + json);
            }
            List<Map<String, Object>> page = JsonPath.read(json, "$.items");
            if (page.size() > size) {
                throw new IllegalStateException("page larger than size");
            }
            items.addAll(page);
            if (!(Boolean) JsonPath.read(json, "$.hasNext")) {
                return items;
            }
            cursor = JsonPath.read(json, "$.nextCursor");
        }
        throw new IllegalStateException("pagination did not terminate");
    }

    /** The tweets inside timeline feed items ({@code {kind, tweet, retweet}}). */
    @SuppressWarnings("unchecked")
    public static List<Map<String, Object>> tweets(List<Map<String, Object>> feedItems) {
        return feedItems.stream().map(item -> (Map<String, Object>) item.get("tweet")).toList();
    }

    public static long id(Map<String, Object> item) {
        return ((Number) item.get("id")).longValue();
    }

    public static String body(org.springframework.mock.web.MockHttpServletResponse response) {
        try {
            return response.getContentAsString(StandardCharsets.UTF_8);
        } catch (Exception e) {
            throw new IllegalStateException(e);
        }
    }
}
