from uuid import UUID, uuid4
from pydantic import BaseModel, Field

class AgentState(BaseModel):
    session_id: UUID = Field(default_factory=uuid4)
    question: str
    current_query: str = ""
    search_count: int = 0
    internal_context: str = ""
    web_context: str = ""
    is_sufficient: bool = False
    draft_answer: str = ""
    critique: str = ""
    revision_count: int = 0
    final_answer: str = ""
    is_consistent: bool = False
    max_results: int = 3
    current_step_message: str = "Инициализация..."
    error: str | None = None