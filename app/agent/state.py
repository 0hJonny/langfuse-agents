# agent/state.py
import operator
from typing import Annotated, List, TypedDict
from shared_types.constants import StepCode

class AgentState(TypedDict):
    session_id: str = ""
    question: str = ""
    current_query: str = ""
    search_count: int = 0
    internal_context: Annotated[List[str], operator.add] 
    web_context: Annotated[List[str], operator.add]
    intent: str = ""
    is_sufficient: bool = False
    draft_answer: str = ""
    critique: str = ""
    revision_count: int = 0
    final_answer: str = ""
    is_consistent: bool = False
    max_results: int = 3
    
    current_step_message: str = StepCode.INIT.value 
    error: str | None = None