-- +goose Up
CREATE TABLE cluster_sources (
    cluster_id INTEGER NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    PRIMARY KEY (cluster_id, source_id)
);

CREATE INDEX idx_cluster_sources_cluster ON cluster_sources(cluster_id);

-- +goose Down
DROP TABLE cluster_sources;
