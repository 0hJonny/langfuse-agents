from contextlib import asynccontextmanager
import asyncio
import logging
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from api.router import api_v1_router 
from api.health import router as health_router
from utils.logging_config import setup_logging
from storage.postgres_saver import create_postgres_checkpointer
from storage.chroma_client import get_chroma_vectorstore
from core.graph import init_agent_app
from core.config import settings
from core.postgres import db_pool

logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    setup_logging()
    logger.info("Запуск приложения...")
    
    # Ленивая проверка Chroma
    try:
        await asyncio.to_thread(get_chroma_vectorstore)
        logger.info("Chroma доступна.")
    except Exception as e:
        logger.error(f"Ошибка подключения к Chroma: {e}")
        raise

    # 1. Открываем глобальный пул PostgreSQL
    await db_pool.open()
    logger.info("Глобальный пул PostgreSQL открыт.")

    try:
        # 2. Передаем пул в чекпоинтер и граф
        async with create_postgres_checkpointer() as checkpointer:
            app.state.graph = await init_agent_app(checkpointer)
            logger.info("Граф и БД готовы к работе.")
            yield
    finally:
        # 3. Закрываем пул при завершении работы сервера
        await db_pool.close()
        logger.info("Завершение работы, пулы закрыты.")

app = FastAPI(lifespan=lifespan)

origins = [o.strip() for o in settings.cors_origins.split(",") if o.strip()]
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(health_router)
app.include_router(api_v1_router)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("main:app", host="0.0.0.0", port=8000, reload=True)