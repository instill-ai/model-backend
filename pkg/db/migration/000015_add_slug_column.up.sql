-- Migration: Add slug and aliases columns for AIP standard
-- Slug is a URL-friendly identifier without prefix, derived from display_name
-- Aliases stores previous slugs for backward compatibility

BEGIN;

-- Add slug column to model table
ALTER TABLE model
ADD COLUMN IF NOT EXISTS slug VARCHAR(255);

-- Add aliases column to model table (stores previous slugs)
ALTER TABLE model
ADD COLUMN IF NOT EXISTS aliases TEXT[];

-- Add display_name column if not exists (some older schemas may not have it)
ALTER TABLE model
ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);

-- Populate display_name from id for existing records if empty
UPDATE model
SET display_name = id
WHERE display_name IS NULL OR display_name = '';

-- Generate slug from display_name for existing model records
-- Slug is URL-safe: lowercase, alphanumeric with hyphens only
UPDATE model
SET slug = LOWER(
    REGEXP_REPLACE(
        REGEXP_REPLACE(
            REGEXP_REPLACE(COALESCE(display_name, id), '[^a-zA-Z0-9\s-]', '', 'g'),
            '\s+',
            '-',
            'g'
        ),
        '-+',
        '-',
        'g'
    )
)
WHERE slug IS NULL OR slug = '';

-- Create index for slug lookups (performance optimization)
CREATE INDEX IF NOT EXISTS idx_model_slug ON model(slug);

COMMIT;
