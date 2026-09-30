-- The classifier labels forced-source decisions "force" (formerly "feed_map"),
-- but the check constraint still only allowed the old name, so every article
-- from a forced-category source (source_force_category.json) was rejected on
-- insert. Accept both: "feed_map" keeps any historical rows valid.

-- +goose Up
ALTER TABLE articles DROP CONSTRAINT IF EXISTS chk_articles_category_method;
ALTER TABLE articles ADD CONSTRAINT chk_articles_category_method CHECK (
    category_method IN ('force', 'feed_map', 'rss_category', 'url_rule', 'keyword_rule', 'fallback')
);

-- +goose Down
DELETE FROM articles WHERE category_method = 'force';
ALTER TABLE articles DROP CONSTRAINT IF EXISTS chk_articles_category_method;
ALTER TABLE articles ADD CONSTRAINT chk_articles_category_method CHECK (
    category_method IN ('feed_map', 'rss_category', 'url_rule', 'keyword_rule', 'fallback')
);
