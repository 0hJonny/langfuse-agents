from chromadb import HttpClient
from chromadb.config import Settings as ChromaSettings
from langchain_chroma import Chroma
from langchain_huggingface import HuggingFaceEmbeddings
from functools import lru_cache
import asyncio
from core.config import settings

_embeddings = HuggingFaceEmbeddings(model_name=settings.embedding_model)

@lru_cache(maxsize=1)
def get_chroma_vectorstore() -> Chroma:
    client = HttpClient(
        host=settings.chroma_host,
        port=settings.chroma_port,
        settings=ChromaSettings(anonymized_telemetry=False)
    )
    return Chroma(
        client=client,
        collection_name=settings.chroma_collection,
        embedding_function=_embeddings,
    )

async def async_similarity_search(query: str, k: int = 3):
    vectorstore = get_chroma_vectorstore()
    return await asyncio.to_thread(vectorstore.similarity_search_with_score, query, k)