package com.fei.twitter.media;

import com.fei.twitter.config.AppProperties;
import java.time.Duration;
import java.util.Optional;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.DisposableBean;
import org.springframework.stereotype.Component;
import software.amazon.awssdk.regions.Region;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.PutObjectRequest;
import software.amazon.awssdk.services.s3.presigner.S3Presigner;

/**
 * The media bucket. Browsers upload straight to S3 with presigned URLs; objects are
 * served from CloudFront when CLOUDFRONT_DOMAIN is set. Without S3_BUCKET_NAME (local
 * dev) uploads fail with a clear error and deletes are no-ops.
 */
@Component
public class MediaStorage implements DisposableBean {

    private static final Logger log = LoggerFactory.getLogger(MediaStorage.class);
    private static final Duration PRESIGN_EXPIRY = Duration.ofMinutes(10);

    private final String bucket;
    private final String baseUrl;
    private final S3Client s3;
    private final S3Presigner presigner;

    MediaStorage(AppProperties props) {
        this.bucket = props.s3Bucket();
        if (bucket == null || bucket.isBlank()) {
            log.warn("S3 bucket is not set; uploads will fail");
            this.baseUrl = "";
            this.s3 = null;
            this.presigner = null;
            return;
        }
        Region region = Region.of(props.s3Region());
        String cdn = props.cloudfrontDomain() == null ? "" : props.cloudfrontDomain().replaceAll("/+$", "");
        this.baseUrl = cdn.isEmpty() ? "https://%s.s3.%s.amazonaws.com".formatted(bucket, region.id()) : "https://" + cdn;
        this.s3 = S3Client.builder().region(region).build();
        this.presigner = S3Presigner.builder().region(region).build();
    }

    /** A URL the browser can PUT exactly this content type and length to. */
    String presignUpload(String objectKey, String contentType, long contentLength) {
        if (presigner == null) {
            throw new IllegalStateException("S3 storage is not configured (S3_BUCKET_NAME)");
        }
        PutObjectRequest put = PutObjectRequest.builder()
                .bucket(bucket).key(objectKey).contentType(contentType).contentLength(contentLength).build();
        return presigner.presignPutObject(p -> p.signatureDuration(PRESIGN_EXPIRY).putObjectRequest(put)).url().toString();
    }

    /** Best effort: a failure is logged, never thrown (the database is the source of truth). */
    public void delete(String objectKey) {
        if (s3 == null || objectKey == null || objectKey.isEmpty()) {
            return;
        }
        try {
            s3.deleteObject(d -> d.bucket(bucket).key(objectKey));
        } catch (RuntimeException e) {
            log.warn("failed to delete S3 object {}", objectKey, e);
        }
    }

    public String publicUrl(String objectKey) {
        return baseUrl.isEmpty() ? objectKey : baseUrl + "/" + objectKey;
    }

    /** The object key of a URL in our bucket; empty for external URLs (e.g. Google avatars). */
    public Optional<String> keyFromPublicUrl(String url) {
        String prefix = baseUrl + "/";
        if (url == null || baseUrl.isEmpty() || !url.startsWith(prefix) || url.length() == prefix.length()) {
            return Optional.empty();
        }
        return Optional.of(url.substring(prefix.length()));
    }

    @Override
    public void destroy() {
        if (s3 != null) {
            s3.close();
            presigner.close();
        }
    }
}
