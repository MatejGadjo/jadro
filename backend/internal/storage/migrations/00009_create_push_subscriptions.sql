-- +goose Up
CREATE TABLE push_subscriptions (
    id           SERIAL PRIMARY KEY,
    endpoint     TEXT NOT NULL UNIQUE,
    p256dh       TEXT NOT NULL,
    auth         TEXT NOT NULL,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    last_sent_at TIMESTAMP WITH TIME ZONE
);

-- push_sends records each daily send so the job can avoid pushing the same
-- cluster two days in a row (a slow story can stay #1). cluster_id is NULL for
-- digest-fallback sends. Durable (survives a Redis flush, unlike a cache key).
CREATE TABLE push_sends (
    id         SERIAL PRIMARY KEY,
    cluster_id BIGINT,
    sent_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_sends_cluster_sent ON push_sends (cluster_id, sent_at DESC);

-- +goose Down
DROP TABLE push_sends;
DROP TABLE push_subscriptions;
