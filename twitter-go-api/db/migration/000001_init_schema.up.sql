CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(30),
    bio VARCHAR(160),
    avatar_url TEXT,
    banner_url TEXT,
    role VARCHAR(20) NOT NULL DEFAULT 'USER',
    provider VARCHAR(50) NOT NULL,
    followers_count INT NOT NULL DEFAULT 0,
    following_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username_trgm ON users USING GIN (username gin_trgm_ops);
CREATE INDEX idx_users_displayname_trgm ON users USING GIN (display_name gin_trgm_ops);

CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expiry_date TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tweets (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content VARCHAR(280),
    media_type VARCHAR(20),
    media_url TEXT,
    parent_id BIGINT REFERENCES tweets(id) ON DELETE CASCADE,
    retweet_id BIGINT REFERENCES tweets(id) ON DELETE CASCADE,
    reply_count INT NOT NULL DEFAULT 0,
    retweet_count INT NOT NULL DEFAULT 0,
    like_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_tweet_has_content_or_media CHECK (content IS NOT NULL OR media_url IS NOT NULL OR retweet_id IS NOT NULL)
);

ALTER TABLE tweets
ADD COLUMN search_vector tsvector
GENERATED ALWAYS AS (to_tsvector('english', COALESCE(content, ''))) STORED;

CREATE UNIQUE INDEX idx_unique_user_retweet ON tweets(user_id, retweet_id) WHERE retweet_id IS NOT NULL;
CREATE INDEX idx_tweets_user_created ON tweets(user_id, created_at DESC);
CREATE INDEX idx_tweets_parent ON tweets(parent_id);
CREATE INDEX idx_tweets_retweet ON tweets(retweet_id);
CREATE INDEX idx_tweets_created_at ON tweets(created_at DESC);
CREATE INDEX idx_tweets_search ON tweets USING GIN(search_vector);
CREATE INDEX idx_tweets_toplevel_created ON tweets(created_at DESC) WHERE parent_id IS NULL;

CREATE TABLE follows (
    follower_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    following_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (follower_id, following_id),
    CHECK (follower_id != following_id)
);

CREATE INDEX idx_follows_follower ON follows(follower_id);
CREATE INDEX idx_follows_following ON follows(following_id);

CREATE TABLE tweet_likes (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tweet_id BIGINT NOT NULL REFERENCES tweets(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, tweet_id)
);

CREATE INDEX idx_tweet_likes_tweet ON tweet_likes(tweet_id);

CREATE TABLE hashtags (
    id BIGSERIAL PRIMARY KEY,
    text VARCHAR(100) NOT NULL UNIQUE,
    usage_count INT NOT NULL DEFAULT 1,
    last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_hashtags_text ON hashtags(text);

CREATE TABLE tweet_hashtags (
    tweet_id BIGINT NOT NULL REFERENCES tweets(id) ON DELETE CASCADE,
    hashtag_id BIGINT NOT NULL REFERENCES hashtags(id) ON DELETE CASCADE,
    PRIMARY KEY (tweet_id, hashtag_id)
);

CREATE INDEX idx_tweet_hashtags_tag ON tweet_hashtags(hashtag_id);

CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    recipient_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tweet_id BIGINT REFERENCES tweets(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_recipient ON notifications(recipient_id, created_at DESC);
CREATE INDEX idx_notifications_unread ON notifications(recipient_id) WHERE is_read = FALSE;
-- Re-liking or re-following must not create a second notification for the same event.
CREATE UNIQUE INDEX idx_notifications_dedupe ON notifications(recipient_id, actor_id, type, tweet_id) NULLS NOT DISTINCT;

CREATE TABLE conversations (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE conversation_participants (
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX idx_conversation_participants_user ON conversation_participants(user_id, conversation_id);

CREATE TABLE direct_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content VARCHAR(2000) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_direct_messages_conversation_created ON direct_messages(conversation_id, created_at DESC, id DESC);

CREATE TABLE tweet_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tweet_id BIGINT UNIQUE NOT NULL REFERENCES tweets(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    embedding vector(768) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tweet_embeddings_embedding ON tweet_embeddings USING hnsw (embedding vector_cosine_ops);
