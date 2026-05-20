-- +goose Up
CREATE TABLE cluster_vectors (
    cluster_id BIGINT PRIMARY KEY REFERENCES clusters(id) ON DELETE CASCADE,
    vector vector(384) NOT NULL,
    article_count INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE INDEX idx_cluster_vectors_vector
    ON cluster_vectors
    USING ivfflat (vector vector_cosine_ops)
    WITH (lists = 100);

-- +goose Down
DROP TABLE cluster_vectors;
