# queries/chat.py

ENSURE_SESSION = """
    INSERT INTO chats.sessions (id, user_id, guest_id, title)
    VALUES (%s, %s, %s, %s)
    ON CONFLICT (id) DO NOTHING
"""

INSERT_MESSAGE = """
    INSERT INTO chats.messages (session_id, parent_id, role, content, trace_id, meta_data)
    VALUES (%s, %s, %s, %s, %s, %s)
    RETURNING id
"""

FIND_MESSAGE_ID_BY_TRACE = """
    SELECT id FROM chats.messages WHERE trace_id = %s
"""

UPSERT_FEEDBACK = """
    INSERT INTO chats.feedback (message_id, rating, comment)
    VALUES (%s, %s, %s)
    ON CONFLICT (message_id) DO UPDATE 
    SET rating = EXCLUDED.rating, comment = EXCLUDED.comment
"""