-- Add pipeline_stages column to messages table
ALTER TABLE messages ADD COLUMN pipeline_stages JSONB DEFAULT '{}';