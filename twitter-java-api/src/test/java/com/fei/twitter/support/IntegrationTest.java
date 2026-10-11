package com.fei.twitter.support;

import java.lang.annotation.ElementType;
import java.lang.annotation.Retention;
import java.lang.annotation.RetentionPolicy;
import java.lang.annotation.Target;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.context.annotation.Import;

/** A full application against a real Postgres (Testcontainers), driven through MockMvc. */
@Target(ElementType.TYPE)
@Retention(RetentionPolicy.RUNTIME)
@SpringBootTest(properties = {
        "TOKEN_SYMMETRIC_KEY=test-key-0123456789-0123456789-0123456789",
        "GOOGLE_CLIENT_ID=test-client",
        "ALLOWED_ORIGINS=http://localhost:3000",
        "spring.config.import=",
        "RATE_LIMITS_ENABLED=false",
})
@AutoConfigureMockMvc
@Import({TestcontainersConfig.class, FakeGoogle.class})
public @interface IntegrationTest {}
