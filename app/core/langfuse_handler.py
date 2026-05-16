from langfuse.langchain import CallbackHandler
from core.config import settings

def get_langfuse_handler() -> CallbackHandler | None:
    if settings.langfuse_public_key and settings.langfuse_secret_key:
        return CallbackHandler()
    return None