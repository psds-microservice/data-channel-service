CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS channel_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL,
  user_id UUID NOT NULL,
  kind VARCHAR(32) NOT NULL DEFAULT 'chat',
  payload JSONB NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_channel_messages_session_id ON channel_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_channel_messages_created_at ON channel_messages(session_id, created_at);
