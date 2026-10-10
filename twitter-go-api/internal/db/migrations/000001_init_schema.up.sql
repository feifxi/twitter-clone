CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    username        VARCHAR(20)  NOT NULL CHECK (username ~ '^[A-Za-z0-9_]{3,20}$'),
    email           VARCHAR(254) NOT NULL UNIQUE,
    display_name    VARCHAR(50),
    bio             VARCHAR(160),
    location        VARCHAR(30),
    website         VARCHAR(100),
    avatar_url      TEXT,
    banner_url      TEXT,
    role            VARCHAR(20)  NOT NULL DEFAULT 'USER',
    provider        VARCHAR(50)  NOT NULL,
    followers_count INT          NOT NULL DEFAULT 0,
    following_count INT          NOT NULL DEFAULT 0,
    -- NULL until the user has picked their own username (sign-up assigns a generated one).
    onboarded_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Usernames are unique regardless of case: "Alice" and "alice" are the same handle.
CREATE UNIQUE INDEX idx_users_username_lower ON users (lower(username));
CREATE INDEX idx_users_username_trgm ON users USING GIN (username gin_trgm_ops);
CREATE INDEX idx_users_displayname_trgm ON users USING GIN (display_name gin_trgm_ops);

-- One row per signed-in device/session; rotated on every refresh.
CREATE TABLE refresh_tokens (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,
    expiry_date TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);

CREATE TABLE tweets (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content       VARCHAR(280),
    media_type    VARCHAR(10) CHECK (media_type IN ('IMAGE', 'VIDEO')),
    media_url     TEXT,
    parent_id     BIGINT      REFERENCES tweets(id) ON DELETE CASCADE,
    reply_count   INT         NOT NULL DEFAULT 0,
    retweet_count INT         NOT NULL DEFAULT 0,
    like_count    INT         NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector tsvector GENERATED ALWAYS AS (to_tsvector('english', COALESCE(content, ''))) STORED,
    CONSTRAINT chk_tweet_has_content_or_media CHECK (content IS NOT NULL OR media_url IS NOT NULL),
    CONSTRAINT chk_tweet_media_pair CHECK ((media_url IS NULL) = (media_type IS NULL))
);

-- Keyset pagination orders by (created_at, id); these indexes match that order.
CREATE INDEX idx_tweets_user_created ON tweets(user_id, created_at DESC, id DESC);
CREATE INDEX idx_tweets_user_media ON tweets(user_id, created_at DESC, id DESC) WHERE media_url IS NOT NULL;
CREATE INDEX idx_tweets_parent_created ON tweets(parent_id, created_at, id);
CREATE INDEX idx_tweets_toplevel_created ON tweets(created_at DESC, id DESC) WHERE parent_id IS NULL;
CREATE INDEX idx_tweets_search ON tweets USING GIN(search_vector);

CREATE TABLE retweets (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tweet_id   BIGINT      NOT NULL REFERENCES tweets(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, tweet_id)
);

CREATE INDEX idx_retweets_user_created ON retweets(user_id, created_at DESC, id DESC);
CREATE INDEX idx_retweets_tweet ON retweets(tweet_id);

CREATE TABLE follows (
    follower_id  BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, following_id),
    CHECK (follower_id <> following_id)
);

CREATE INDEX idx_follows_follower_created ON follows(follower_id, created_at DESC, following_id DESC);
CREATE INDEX idx_follows_following_created ON follows(following_id, created_at DESC, follower_id DESC);

CREATE TABLE tweet_likes (
    user_id    BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tweet_id   BIGINT      NOT NULL REFERENCES tweets(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, tweet_id)
);

CREATE INDEX idx_tweet_likes_tweet ON tweet_likes(tweet_id);
CREATE INDEX idx_tweet_likes_user_created ON tweet_likes(user_id, created_at DESC, tweet_id DESC);

CREATE TABLE hashtags (
    id           BIGSERIAL PRIMARY KEY,
    text         VARCHAR(100) NOT NULL UNIQUE,
    usage_count  INT          NOT NULL DEFAULT 1,
    last_used_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE tweet_hashtags (
    tweet_id   BIGINT NOT NULL REFERENCES tweets(id) ON DELETE CASCADE,
    hashtag_id BIGINT NOT NULL REFERENCES hashtags(id) ON DELETE CASCADE,
    PRIMARY KEY (tweet_id, hashtag_id)
);

CREATE INDEX idx_tweet_hashtags_tag ON tweet_hashtags(hashtag_id, tweet_id);

CREATE TABLE notifications (
    id           BIGSERIAL PRIMARY KEY,
    recipient_id BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tweet_id     BIGINT      REFERENCES tweets(id) ON DELETE CASCADE,
    type         VARCHAR(20) NOT NULL CHECK (type IN ('REPLY', 'LIKE', 'RETWEET', 'FOLLOW')),
    is_read      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_recipient ON notifications(recipient_id, created_at DESC, id DESC);
CREATE INDEX idx_notifications_unread ON notifications(recipient_id) WHERE is_read = FALSE;
-- Re-liking or re-following must not create a second notification for the same event.
CREATE UNIQUE INDEX idx_notifications_dedupe ON notifications(recipient_id, actor_id, type, tweet_id) NULLS NOT DISTINCT;

CREATE TABLE conversations (
    id         BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE conversation_participants (
    conversation_id BIGINT      NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conversation_participants_user ON conversation_participants(user_id, conversation_id);

CREATE TABLE direct_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content         VARCHAR(2000) NOT NULL,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_direct_messages_conversation_created ON direct_messages(conversation_id, created_at DESC, id DESC);

CREATE TABLE tweet_embeddings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tweet_id   BIGINT      NOT NULL UNIQUE REFERENCES tweets(id) ON DELETE CASCADE,
    content    TEXT        NOT NULL,
    embedding  vector(768) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tweet_embeddings_embedding ON tweet_embeddings USING hnsw (embedding vector_cosine_ops);

-- Tweets waiting to be embedded for RAG. Enqueued in the same transaction as the
-- tweet (no lost events); workers lease rows with FOR UPDATE SKIP LOCKED.
CREATE TABLE embedding_jobs (
    tweet_id   BIGINT      PRIMARY KEY REFERENCES tweets(id) ON DELETE CASCADE,
    attempts   INT         NOT NULL DEFAULT 0,
    run_after  TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_embedding_jobs_due ON embedding_jobs(run_after);

-- "For You" ranking: engagement with time decay, evaluated at a fixed anchor time
-- so a page's ordering doesn't drift while the user scrolls.
CREATE FUNCTION tweet_score(likes INT, retweets INT, replies INT, created_at TIMESTAMPTZ, anchor TIMESTAMPTZ)
RETURNS FLOAT8
LANGUAGE sql IMMUTABLE
AS $$
    SELECT (likes * 2 + retweets * 3 + replies + 1)::float8
         / POWER(GREATEST(EXTRACT(EPOCH FROM anchor - created_at), 0) / 3600 + 2, 1.8)
$$;
