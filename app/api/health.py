# api/health.py
import logging
import asyncio
from fastapi import APIRouter, HTTPException
from psycopg import AsyncConnection
from chromadb import HttpClient
from core.config import settings

logger = logging.getLogger(__name__)
router = APIRouter(tags=["health"])

async def check_postgres():
    """Проверка Postgres с таймаутом."""
    try:
        async with asyncio.timeout(3.0): # Ждем максимум 3 секунды
            async with await AsyncConnection.connect(settings.postgres_uri) as conn:
                async with conn.cursor() as cur:
                    await cur.execute("SELECT 1")
    except Exception as e:
        logger.error(f"PostgreSQL health check failed: {e}")
        raise HTTPException(status_code=503, detail="PostgreSQL unavailable")

async def check_chroma():
    """Проверка ChromaDB (асинхронно через thread)."""
    try:
        def _ping():
            client = HttpClient(host=settings.chroma_host, port=settings.chroma_port)
            client.heartbeat()
            
        async with asyncio.timeout(3.0):
            await asyncio.to_thread(_ping)
    except Exception as e:
        logger.error(f"Chroma health check failed: {e}")
        raise HTTPException(status_code=503, detail="Chroma unavailable")

@router.get("/health")
async def health_check():
    # Запускаем проверки параллельно
    await asyncio.gather(check_postgres(), check_chroma())
    return {"status": "ok", "postgres": "connected", "chroma": "connected"}