from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    llm_provider: str = "ollama"
    ollama_base_url: str = "http://localhost:11434/v1"
    ollama_model: str = "gemma2:9b"
    lmstudio_base_url: str = "http://host.docker.internal:1234/v1"
    lmstudio_model: str = "google/gemma-4-e4b"
    llm_temperature: float = 0.0

    embedding_model: str = "all-MiniLM-L6-v2"

    chroma_host: str = "localhost"
    chroma_port: int = 8000
    chroma_collection: str = "local_it_news"

    postgres_uri: str = "postgresql://agent_user:agent_pass@postgres:5432/agent_db"

    langfuse_public_key: str | None = None
    langfuse_secret_key: str | None = None
    langfuse_host: str = "https://cloud.langfuse.com"

    cors_origins: str = "http://localhost:3000"
    max_context_tokens: int = 3000

    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

settings = Settings()