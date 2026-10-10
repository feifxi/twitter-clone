package com.fei.twitter.config;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.Map;
import org.junit.jupiter.api.Test;
import org.springframework.core.env.MapPropertySource;
import org.springframework.core.env.StandardEnvironment;

class GoStyleEnvironmentTest {

    @Test
    void translatesTheGoApiVariables() {
        StandardEnvironment env = new StandardEnvironment();
        env.getPropertySources().addFirst(new MapPropertySource("app.env", Map.of(
                "DATABASE_URL", "postgresql://root:p%40ss@db.internal:5432/twitter_db?sslmode=require",
                "HTTP_SERVER_ADDRESS", "0.0.0.0:9090",
                "REDIS_URL", "redis://redis:6379",
                "LOG_LEVEL", "warn",
                "ENVIRONMENT", "production")));

        new GoStyleEnvironment().postProcessEnvironment(env, null);

        assertThat(env.getProperty("spring.datasource.url")).isEqualTo("jdbc:postgresql://db.internal:5432/twitter_db?sslmode=require");
        assertThat(env.getProperty("spring.datasource.username")).isEqualTo("root");
        assertThat(env.getProperty("spring.datasource.password")).isEqualTo("p@ss");
        assertThat(env.getProperty("server.address")).isEqualTo("0.0.0.0");
        assertThat(env.getProperty("server.port")).isEqualTo("9090");
        assertThat(env.getProperty("spring.data.redis.url")).isEqualTo("redis://redis:6379");
        assertThat(env.getProperty("logging.level.root")).isEqualTo("WARN");
        assertThat(env.getProperty("logging.structured.format.console")).isEqualTo("logstash");
    }

    @Test
    void acceptsThePostgresSchemeAndNoCredentials() {
        assertThat(GoStyleEnvironment.jdbcProperties("postgres://localhost/twitter_db"))
                .containsEntry("spring.datasource.url", "jdbc:postgresql://localhost/twitter_db")
                .doesNotContainKey("spring.datasource.username");
    }
}
