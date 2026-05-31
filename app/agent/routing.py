import logging
from .state import AgentState

logger = logging.getLogger(__name__)

def route_after_classification(state: AgentState) -> str:
    if state.get("intent") in ["chitchat", "capabilities"]:
        return "simple_response"
    return "internal_search"

def route_after_evaluation(state: AgentState) -> str:
    if state.get("is_sufficient"):
        return "synthesize"

    # Если ещё не исчерпали лимит поисков (2 попытки)
    if state.get("search_count") < 2:
        return "reformulate"

    logger.info("Search limit reached, proceeding to synthesis with available data.")
    return "synthesize"


def route_after_fact_check(state: AgentState) -> str:
    if state.get("is_consistent") or state.get("revision_count") >= 2:
        return "finalize"
    logger.info("Fact check failed, sending back to writer for revision.")
    return "synthesize"