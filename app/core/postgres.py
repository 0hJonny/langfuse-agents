from psycopg_pool import AsyncConnectionPool
from core.config import settings

db_pool = AsyncConnectionPool(
    conninfo=settings.postgres_uri,
    max_size=20,
    kwargs={"autocommit": True, "prepare_threshold": 0},
    open=False,
)