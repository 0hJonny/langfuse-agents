from contextlib import asynccontextmanager
import logging
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from api.routes import router as chat_router
from api.feedback import router as feedback_router
from api.health import router as health_router
from utils.logging_config import setup_logging
from storage.postgres_saver import create_postgres_checkpointer
from storage.chroma_client import get_chroma_vectorstore
from core.graph import compile_graph
from core.config import settings

logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    setup_logging()
    logger.info("Запуск приложения...")
    
    # Проверяем Chroma
    try:
        get_chroma_vectorstore()
        logger.info("Chroma доступна.")
    except Exception as e:
        logger.error(f"Ошибка подключения к Chroma: {e}")
        raise

    # Управляем жизненным циклом PostgreSQL и графа
    async with create_postgres_checkpointer() as checkpointer:
        app.state.graph = await compile_graph(checkpointer)
        logger.info("Граф и БД готовы к работе.")
        yield
        logger.info("Завершение работы, закрытие соединения с БД...")

app = FastAPI(lifespan=lifespan)

# Настройка CORS
origins = [o.strip() for o in settings.cors_origins.split(",") if o.strip()]
app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(chat_router)
app.include_router(feedback_router)
app.include_router(health_router)

if __name__ == "__main__":
    import uvicorn
    uvicorn.run("main:app", host="0.0.0.0", port=8000, reload=True)