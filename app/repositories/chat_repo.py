import json
from uuid import UUID
from psycopg_pool import AsyncConnectionPool
from queries import chat as chat_queries

class ChatRepository:
    def __init__(self, pool: AsyncConnectionPool):
        self.pool = pool

    async def ensure_session(self, session_id: UUID, user_id: str = None, guest_id: str = None, title: str = "Новый чат"):
        async with self.pool.connection() as conn:
            await conn.execute(
                chat_queries.ENSURE_SESSION, 
                (str(session_id), user_id, guest_id, title)
            )

    async def save_message(self, session_id: UUID, role: str, content: str, 
                           trace_id: str = None, parent_id: UUID = None, meta_data: dict = None) -> UUID:
        async with self.pool.connection() as conn:
            result = await conn.execute(
                chat_queries.INSERT_MESSAGE, 
                (
                    str(session_id), 
                    str(parent_id) if parent_id else None,
                    role, 
                    content, 
                    trace_id, 
                    json.dumps(meta_data or {})
                )
            )
            row = await result.fetchone()
            return row[0]

    async def save_feedback_by_trace(self, trace_id: str, rating: str, comment: str = None) -> bool:
        async with self.pool.connection() as conn:
            # Ищем ID сообщения
            res = await conn.execute(chat_queries.FIND_MESSAGE_ID_BY_TRACE, (trace_id,))
            row = await res.fetchone()
            
            if row:
                message_id = row[0]
                # Сохраняем или обновляем фидбек
                await conn.execute(
                    chat_queries.UPSERT_FEEDBACK, 
                    (message_id, rating, comment)
                )
                return True
            return False