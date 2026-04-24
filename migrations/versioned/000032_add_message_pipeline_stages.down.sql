-- Remove pipeline_stages column from messages table
ALTER TABLE messages DROP COLUMN IF EXISTS pipeline_stages;