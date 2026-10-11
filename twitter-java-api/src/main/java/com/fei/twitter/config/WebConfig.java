package com.fei.twitter.config;

import com.fei.twitter.security.ViewerArgumentResolver;
import com.fei.twitter.shared.ratelimit.RateLimitInterceptor;
import com.fei.twitter.shared.web.PageRequestArgumentResolver;
import java.time.Clock;
import java.time.Duration;
import java.util.List;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.CorsConfigurationSource;
import org.springframework.web.cors.UrlBasedCorsConfigurationSource;
import org.springframework.web.method.support.HandlerMethodArgumentResolver;
import org.springframework.web.servlet.config.annotation.InterceptorRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

@Configuration
public class WebConfig implements WebMvcConfigurer {

    private final PageRequestArgumentResolver pageRequestResolver;
    private final ViewerArgumentResolver viewerResolver;
    private final RateLimitInterceptor rateLimits;

    WebConfig(PageRequestArgumentResolver pageRequestResolver, ViewerArgumentResolver viewerResolver, RateLimitInterceptor rateLimits) {
        this.pageRequestResolver = pageRequestResolver;
        this.viewerResolver = viewerResolver;
        this.rateLimits = rateLimits;
    }

    @Override
    public void addArgumentResolvers(List<HandlerMethodArgumentResolver> resolvers) {
        resolvers.add(pageRequestResolver);
        resolvers.add(viewerResolver);
    }

    @Override
    public void addInterceptors(InterceptorRegistry registry) {
        registry.addInterceptor(rateLimits);
    }

    /** Browser origins allowed to call the API with cookies (used by Spring Security's CORS filter). */
    @Bean
    CorsConfigurationSource corsConfigurationSource(AppProperties props) {
        CorsConfiguration cors = new CorsConfiguration();
        cors.setAllowedOrigins(props.allowedOrigins().stream().map(String::trim).toList());
        cors.setAllowedMethods(List.of("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"));
        cors.setAllowedHeaders(List.of("Origin", "Content-Type", "Accept", "Authorization"));
        cors.setExposedHeaders(List.of("Content-Length", "X-Request-ID"));
        cors.setAllowCredentials(true);
        cors.setMaxAge(Duration.ofHours(12));
        UrlBasedCorsConfigurationSource source = new UrlBasedCorsConfigurationSource();
        source.registerCorsConfiguration("/**", cors);
        return source;
    }

    @Bean
    Clock clock() {
        return Clock.systemUTC();
    }
}
