import json
import logging
from fastapi import APIRouter, Request
from sse_starlette.sse import EventSourceResponse

from api.schemas import ChatRequest, ChatEvent, FinalAnswer
from agent.state import AgentState
from core.config import settings
from core.langfuse_handler import get_langfuse_handler

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1", tags=["chat"])

@router.post("/chat/stream")
async def stream_chat(payload: ChatRequest, request: Request):
    graph = request.app.state.graph
    model_name = settings.ollama_model if settings.llm_provider == "ollama" else settings.lmstudio_model

    initial_state = AgentState(session_id=payload.session_id, question=payload.question)
    config = {"configurable": {"thread_id": str(payload.session_id)}}

    langfuse_handler = get_langfuse_handler()
    if langfuse_handler:
        config["callbacks"] = [langfuse_handler]

    async def event_generator():
        try:
            yield {"event": "status", "data": ChatEvent(node="start", message="Запуск...", model=model_name).model_dump_json()}

            async for update in graph.astream(initial_state, config=config, stream_mode="updates"):
                node_name, state_update = next(iter(update.items()))
                if "error" in state_update:
                    yield {"event": "error", "data": json.dumps({"detail": state_update["error"]})}
                    return
                step_msg = state_update.get("current_step_message", "")
                event = ChatEvent(node=node_name, message=step_msg, model=model_name)
                yield {"event": "node_update", "data": event.model_dump_json()}

            final_state = await graph.aget_state(config)
            final_text = final_state.values.get("final_answer", "Не удалось сформировать ответ.")
            final_answer = FinalAnswer(session_id=payload.session_id, answer=final_text, trace_id=langfuse_handler.last_trace_id)
            yield {"event": "final", "data": final_answer.model_dump_json()}

        except Exception as e:
            logger.exception("Stream error")
            yield {"event": "error", "data": json.dumps({"detail": str(e)})}
        finally:
            if settings.langfuse_public_key and settings.langfuse_secret_key:
                from langfuse import get_client
                get_client().flush()

    return EventSourceResponse(event_generator())