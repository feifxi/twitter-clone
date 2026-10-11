package com.fei.twitter.media;

import com.fei.twitter.config.AppProperties;
import com.fei.twitter.security.Viewer;
import com.fei.twitter.shared.error.ApiException;
import com.fei.twitter.shared.ratelimit.RateLimit;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import jakarta.validation.constraints.Min;
import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Pattern;
import java.util.Locale;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@Tag(name = "Uploads")
@RestController
class UploadController {

    private final MediaStorage storage;
    private final AppProperties props;

    UploadController(MediaStorage storage, AppProperties props) {
        this.storage = storage;
        this.props = props;
    }

    record PresignRequest(
            @NotBlank(message = "this field is required") String contentType,
            @NotBlank(message = "this field is required")
            @Pattern(regexp = "tweets|avatars|banners", message = "must be one of [tweets avatars banners]") String folder,
            @NotNull(message = "this field is required") @Min(value = 1, message = "must be at least 1") Long contentLength) {}

    record PresignResponse(String presignedUrl, String objectKey) {}

    @Operation(summary = "A presigned URL to upload one file directly to S3 (exact Content-Type and Content-Length)")
    @RateLimit(RateLimit.Policy.WRITE)
    @PostMapping("/api/v1/uploads/presign")
    PresignResponse presign(@Valid @RequestBody PresignRequest request, Viewer viewer) {
        String contentType = request.contentType().strip().toLowerCase(Locale.ROOT);
        String extension = MediaKeys.EXTENSIONS.get(contentType);
        if (extension == null) {
            throw ApiException.badRequest("unsupported content type");
        }
        long maxSize = switch (request.folder()) {
            case MediaKeys.AVATARS -> props.maxAvatarBytes();
            case MediaKeys.BANNERS -> props.maxBannerBytes();
            default -> props.maxMediaBytes();
        };
        if (request.contentLength() > maxSize) {
            throw ApiException.badRequest("file too large");
        }
        if (!request.folder().equals(MediaKeys.TWEETS) && !contentType.startsWith("image/")) {
            throw ApiException.badRequest("profile images must be images");
        }

        String key = MediaKeys.newKey(request.folder(), viewer.requireId(), extension);
        return new PresignResponse(storage.presignUpload(key, contentType, request.contentLength()), key);
    }
}
