from contextlib import asynccontextmanager
from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from core.config import settings
from typing import AsyncGenerator

@asynccontextmanager
async def create_postgres_checkpointer() -> AsyncGenerator[AsyncPostgresSaver, None]:
    async with AsyncPostgresSaver.from_conn_string(settings.postgres_uri) as checkpointer:
        await checkpointer.setup()
        yield checkpointer