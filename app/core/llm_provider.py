from functools import lru_cache
from langchain_openai import ChatOpenAI
from langchain_ollama import ChatOllama
from core.config import settings

@lru_cache(maxsize=1)
def get_llm():
    if settings.llm_provider == "ollama":
        return ChatOllama(
            model=settings.ollama_model,
            base_url=settings.ollama_base_url,
            temperature=settings.llm_temperature,
        )

    return ChatOpenAI(
        model=settings.lmstudio_model,
        api_key="lm-studio",
        base_url=settings.lmstudio_base_url,
        temperature=settings.llm_temperature,
    )