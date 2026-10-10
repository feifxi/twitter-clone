package com.fei.twitter.security;

import static org.springframework.http.HttpMethod.GET;
import static org.springframework.http.HttpMethod.POST;

import com.fei.twitter.config.AppProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.config.annotation.web.configurers.AbstractHttpConfigurer;
import org.springframework.security.config.http.SessionCreationPolicy;
import org.springframework.security.oauth2.server.resource.web.authentication.BearerTokenAuthenticationFilter;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.security.web.header.writers.ReferrerPolicyHeaderWriter.ReferrerPolicy;
import org.springframework.web.cors.CorsConfigurationSource;
import org.springframework.web.filter.CorsFilter;
import tools.jackson.databind.json.JsonMapper;

/**
 * Who may call what. Public reads allow anonymous callers (signed-in callers get
 * viewer-specific fields such as isLiked); everything else needs a session.
 */
@Configuration
public class SecurityConfig {

    @Bean
    SecurityFilterChain api(HttpSecurity http, TokenService tokens, ApiAuthenticationEntryPoint entryPoint,
                            AppProperties props, JsonMapper json, CorsConfigurationSource corsConfigurationSource) throws Exception {
        SessionTokenResolver tokenResolver = new SessionTokenResolver();

        http.authorizeHttpRequests(routes -> routes
                        // Operations
                        .requestMatchers(GET, "/healthz", "/readyz", "/metrics", "/swagger/**").permitAll()
                        // Sign-in: these manage the session cookies themselves
                        .requestMatchers(POST, "/api/v1/auth/google", "/api/v1/auth/refresh", "/api/v1/auth/logout").permitAll()
                        // Likes are private to their owner
                        .requestMatchers(GET, "/api/v1/users/*/likes").authenticated()
                        // Public reads
                        .requestMatchers(GET,
                                "/api/v1/users/**", "/api/v1/tweets/**", "/api/v1/feeds/global",
                                "/api/v1/search/**", "/api/v1/discovery/**").permitAll()
                        .requestMatchers("/error").permitAll()
                        // Everything else: /auth/me, writes, following feed, notifications, messages, events, assistant
                        .anyRequest().authenticated())
                .oauth2ResourceServer(oauth -> oauth
                        .bearerTokenResolver(tokenResolver)
                        .authenticationEntryPoint(entryPoint)
                        .jwt(jwt -> jwt.decoder(tokens.decoder())))
                .exceptionHandling(e -> e.authenticationEntryPoint(entryPoint))
                .sessionManagement(s -> s.sessionCreationPolicy(SessionCreationPolicy.STATELESS))
                .cors(AbstractHttpConfigurer::disable) // replaced by the CorsFilter below (JSON rejections)
                // Token-based CSRF doesn't fit a cookie API called from another subdomain;
                // SameSite=Lax cookies plus the Origin check below do the job (as in the Go API).
                .csrf(AbstractHttpConfigurer::disable)
                .addFilterBefore(corsFilter(corsConfigurationSource, json), BearerTokenAuthenticationFilter.class)
                .addFilterBefore(new CrossOriginProtectionFilter(props.allowedOrigins(), json), BearerTokenAuthenticationFilter.class)
                .addFilterBefore(new StaleSessionFilter(tokenResolver, entryPoint), BearerTokenAuthenticationFilter.class)
                .headers(headers -> headers
                        .frameOptions(frame -> frame.deny())
                        .referrerPolicy(referrer -> referrer.policy(ReferrerPolicy.STRICT_ORIGIN_WHEN_CROSS_ORIGIN)))
                .httpBasic(AbstractHttpConfigurer::disable)
                .formLogin(AbstractHttpConfigurer::disable)
                .logout(AbstractHttpConfigurer::disable);
        return http.build();
    }

    private static CorsFilter corsFilter(CorsConfigurationSource config, JsonMapper json) {
        CorsFilter filter = new CorsFilter(config);
        filter.setCorsProcessor(new JsonCorsProcessor(json));
        return filter;
    }
}
