-- Update unique constraints for user-specific URL shortening
-- Remove old unique index on original_url (allows same URL for different users)
DROP INDEX IF EXISTS idx_urls_original_url;

-- Add composite unique index on (original_url, user_id) (prevents duplicate URLs per user)
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_original_url_user_id ON urls(original_url, user_id);
