-- Remove is_deleted column from urls table
DROP INDEX IF EXISTS idx_urls_is_deleted;
ALTER TABLE urls DROP COLUMN is_deleted;
