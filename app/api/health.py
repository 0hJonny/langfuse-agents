import logging
from fastapi import APIRouter, HTTPException
from psycopg import AsyncConnection
from chromadb import HttpClient
from core.config import settings

logger = logging.getLogger(__name__)
router = APIRouter(tags=["health"])

@router.get("/health")
async def health_check():
    """Проверка доступности PostgreSQL и ChromaDB."""
    try:
        async with await AsyncConnection.connect(settings.postgres_uri) as conn:
            async with conn.cursor() as cur:
                await cur.execute("SELECT 1")
    except Exception as e:
        logger.error(f"PostgreSQL health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"PostgreSQL unavailable: {str(e)}")

    try:
        client = HttpClient(host=settings.chroma_host, port=settings.chroma_port)
        client.heartbeat()
    except Exception as e:
        logger.error(f"Chroma health check failed: {e}")
        raise HTTPException(status_code=503, detail=f"Chroma unavailable: {str(e)}")

    return {"status": "ok", "postgres": "connected", "chroma": "connected"}