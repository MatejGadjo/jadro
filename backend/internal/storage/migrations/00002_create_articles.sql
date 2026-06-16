-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TYPE embedding_status AS ENUM ('pending', 'processing', 'ready', 'failed');

CREATE TABLE articles (
    id SERIAL PRIMARY KEY,
    source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    snippet TEXT,
    normalized_title TEXT NOT NULL,
    topic TEXT NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    fetched_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    normalized_snippet TEXT,
    embedding vector(384),
    embedding_status embedding_status NOT NULL DEFAULT 'pending',
    rss_categories TEXT[],
    feed_id INTEGER REFERENCES sources(id) ON DELETE SET NULL,
    feed_url TEXT,
    source_domain TEXT,
    category TEXT NOT NULL,
    category_confidence REAL NOT NULL,
    category_method TEXT NOT NULL,
    category_signals JSONB NOT NULL DEFAULT '{}'::jsonb,
    image_url TEXT,
    CONSTRAINT chk_articles_topic_canonical CHECK (topic IN (
        'Македонија', 'Економија', 'Технологија', 'Свет', 'Скопје',
        'Спорт', 'Хроника', 'Култура', 'Сцена', 'Регион'
    )),
    CONSTRAINT chk_articles_category_canonical CHECK (category IN (
        'Македонија', 'Економија', 'Технологија', 'Свет', 'Скопје',
        'Спорт', 'Хроника', 'Култура', 'Сцена', 'Регион'
    )),
    CONSTRAINT chk_articles_category_confidence_range CHECK (
        category_confidence >= 0 AND category_confidence <= 1
    ),
    CONSTRAINT chk_articles_category_method CHECK (
        category_method IN ('feed_map', 'rss_category', 'url_rule', 'keyword_rule', 'fallback')
    )
);

CREATE INDEX idx_articles_published_at ON articles(published_at);
CREATE INDEX idx_articles_topic ON articles(topic);
CREATE INDEX idx_articles_embedding ON articles USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
CREATE INDEX idx_articles_embedding_status ON articles(embedding_status);

-- +goose Down
DROP TABLE articles;
DROP TYPE embedding_status;
DROP EXTENSION IF EXISTS vector;
