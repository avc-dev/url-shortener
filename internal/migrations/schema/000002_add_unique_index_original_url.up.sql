-- Add unique index on original_url to prevent duplicate URLs
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_urls_original_url ON urls(original_url);
