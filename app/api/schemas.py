from uuid import UUID
from pydantic import BaseModel, Field

class ChatRequest(BaseModel):
    session_id: UUID = Field(..., description="UUID чат-сессии")
    question: str = Field(..., min_length=1, max_length=2000, description="Вопрос пользователя")

class ChatEvent(BaseModel):
    """Событие, отправляемое через SSE."""
    node: str = Field(..., description="Имя текущего узла графа")
    message: str = Field(..., description="Человекочитаемое сообщение о шаге")
    model: str | None = Field(None, description="Используемая LLM модель (если применимо)")

class FinalAnswer(BaseModel):
    session_id: UUID
    answer: str
    trace_id: str | None

class FeedbackRequest(BaseModel):
    trace_id: str = Field(..., description="ID trace сообщения, к которой относится оценка")
    rating: str = Field(..., pattern="^(like|dislike)$", description="Оценка: like или dislike")
    comment: str | None = Field(None, max_length=500, description="Опциональный комментарий")