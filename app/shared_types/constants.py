# shared_types/constants.py
from enum import Enum

class StepCode(str, Enum):
    INIT = "init"
    CLASSIFYING = "classifying_intent"
    INTENT_CHITCHAT = "intent_chitchat"
    INTENT_RAG = "intent_rag"
    INTENT_CAPABILITIES = "intent_capabilities"
    LOCAL_SEARCH = "local_search"
    LOCAL_FOUND = "local_found"
    LOCAL_NOT_FOUND = "local_not_found"
    EVALUATING = "evaluating_context"
    REFORMULATING = "reformulating_query"
    WEB_SEARCH = "web_search"
    WEB_FOUND = "web_found"
    WEB_NOT_FOUND = "web_not_found"
    SYNTHESIZING = "synthesizing_draft"
    FACT_CHECKING = "fact_checking"
    FINALIZING = "finalizing_response"
    ERROR = "error"