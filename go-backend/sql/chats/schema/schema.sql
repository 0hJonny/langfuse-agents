CREATE SCHEMA IF NOT EXISTS chats;

CREATE TYPE chats.feedback_rating AS ENUM (
    'like',
    'dislike'
);

CREATE TYPE chats.message_role AS ENUM (
    'user',
    'assistant',
    'system'
);

CREATE TABLE chats.sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title character varying(255) DEFAULT 'New chat'::character varying NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    deleted_at timestamp with time zone
);

CREATE TABLE chats.messages (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id uuid NOT NULL REFERENCES chats.sessions(id) ON DELETE CASCADE,
    parent_id uuid REFERENCES chats.messages(id) ON DELETE SET NULL,
    role chats.message_role NOT NULL,
    content text NOT NULL,
    trace_id character varying(255),
    meta_data jsonb DEFAULT '{}'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE chats.feedback (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id uuid NOT NULL REFERENCES chats.messages(id) ON DELETE CASCADE,
    rating chats.feedback_rating NOT NULL,
    comment text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE INDEX idx_sessions_user_active ON chats.sessions(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_messages_session_time ON chats.messages(session_id, created_at ASC);
CREATE UNIQUE INDEX idx_feedback_message ON chats.feedback(message_id);
