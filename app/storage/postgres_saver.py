from contextlib import asynccontextmanager
from typing import AsyncGenerator
from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from core.postgres import db_pool

@asynccontextmanager
async def create_postgres_checkpointer() -> AsyncGenerator[AsyncPostgresSaver, None]:
    checkpointer = AsyncPostgresSaver(db_pool)
    await checkpointer.setup()
    yield checkpointer