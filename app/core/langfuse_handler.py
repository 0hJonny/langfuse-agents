import logging
from langfuse import Langfuse
from langfuse.langchain import CallbackHandler
from core.config import settings

logger = logging.getLogger(__name__)

def get_langfuse_handler() -> CallbackHandler | None:
    if not settings.langfuse_public_key or not settings.langfuse_secret_key:
        logger.warning("Ключи Langfuse не настроены. Трейсинг отключен.")
        return None

    try:
        Langfuse(
            public_key=settings.langfuse_public_key,
            secret_key=settings.langfuse_secret_key,
            host=settings.langfuse_host
        )
        
        return CallbackHandler()
        
    except Exception as e:
        logger.error(f"Ошибка инициализации Langfuse: {e}")
        return None