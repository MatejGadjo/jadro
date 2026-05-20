-- +goose Up
CREATE TABLE cluster_articles (
    cluster_id INTEGER NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    article_id INTEGER NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    PRIMARY KEY (cluster_id, article_id)
);

CREATE INDEX idx_cluster_articles_cluster ON cluster_articles(cluster_id);

-- +goose Down
DROP TABLE cluster_articles;
