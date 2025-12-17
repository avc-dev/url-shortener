-- Create urls table for storing shortened URLs
CREATE TABLE IF NOT EXISTS urls (
    id SERIAL PRIMARY KEY,
    code VARCHAR(10) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on code for fast lookups
CREATE INDEX IF NOT EXISTS idx_urls_code ON urls(code);