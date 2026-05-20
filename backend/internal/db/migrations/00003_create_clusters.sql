-- +goose Up
CREATE TABLE clusters (
    id SERIAL PRIMARY KEY,
    headline TEXT NOT NULL,
    summary TEXT,
    topic TEXT NOT NULL,
    score REAL NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    article_count INTEGER NOT NULL DEFAULT 0,
    source_count INTEGER NOT NULL DEFAULT 0,
    topic_confidence REAL,
    display_title TEXT NOT NULL DEFAULT '',
    articles_last_6h INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT chk_clusters_topic_canonical CHECK (topic IN (
        'Македонија', 'Економија', 'Технологија', 'Свет', 'Скопје',
        'Спорт', 'Хроника', 'Култура', 'Сцена', 'Регион'
    ))
);

CREATE INDEX idx_clusters_updated_at ON clusters(updated_at);
CREATE INDEX idx_clusters_topic ON clusters(topic);
CREATE INDEX idx_clusters_last_seen_at ON clusters(last_seen_at);

-- +goose Down
DROP TABLE clusters;
