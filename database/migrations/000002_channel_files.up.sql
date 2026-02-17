CREATE TABLE IF NOT EXISTS channel_files (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL,
  user_id UUID NOT NULL,
  filename VARCHAR(255) NOT NULL,
  content_type VARCHAR(128),
  size_bytes BIGINT DEFAULT 0,
  storage_path VARCHAR(512),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_channel_files_session_id ON channel_files(session_id);
