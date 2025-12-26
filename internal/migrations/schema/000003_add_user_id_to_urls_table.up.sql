-- Add user_id column to urls table
ALTER TABLE urls ADD COLUMN user_id VARCHAR(36) DEFAULT NULL;
