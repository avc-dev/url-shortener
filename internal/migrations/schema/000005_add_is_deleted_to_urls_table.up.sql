-- Add is_deleted column to urls table for soft delete functionality
ALTER TABLE urls ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

-- Create index on is_deleted for efficient filtering
CREATE INDEX IF NOT EXISTS idx_urls_is_deleted ON urls(is_deleted);
