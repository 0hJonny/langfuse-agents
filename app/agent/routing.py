import logging
from .state import AgentState

logger = logging.getLogger(__name__)

def route_after_evaluation(state: AgentState) -> str:
    if state.is_sufficient:
        return "synthesize"

    # Если ещё не исчерпали лимит поисков (2 попытки)
    if state.search_count < 2:
        return "reformulate"

    logger.info("Search limit reached, proceeding to synthesis with available data.")
    return "synthesize"


def route_after_fact_check(state: AgentState) -> str:
    if state.is_consistent or state.revision_count >= 2:
        return "finalize"
    logger.info("Fact check failed, sending back to writer for revision.")
    return "synthesize"