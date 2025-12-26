-- Revert unique constraints update
-- Remove composite unique index
DROP INDEX IF EXISTS idx_urls_original_url_user_id;

-- Restore old unique index on original_url
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url);
