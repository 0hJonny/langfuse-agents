import json
import logging
import asyncio
import uuid
from fastapi import APIRouter, Request, Depends
from sse_starlette.sse import EventSourceResponse
from langgraph.graph.state import CompiledStateGraph

from api.schemas import ChatRequest, ChatEvent, FinalAnswer
from api.dependencies import get_graph, get_current_user
from core.config import settings
from core.langfuse_handler import get_langfuse_handler
from core.postgres import db_pool                
from repositories.chat_repo import ChatRepository 

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/chat", tags=["chat"])

@router.post("/stream")
async def stream_chat(
    payload: ChatRequest, 
    request: Request,
    graph: CompiledStateGraph = Depends(get_graph),
    user_id: str = Depends(get_current_user)
):
    model_name = settings.ollama_model if settings.llm_provider == "ollama" else settings.lmstudio_model
    
    repo = ChatRepository(db_pool)
    
    actual_user_id = None
    actual_guest_id = None
    
    try:
        if user_id:
            uuid.UUID(str(user_id))
            actual_user_id = str(user_id)
    except ValueError:
        actual_guest_id = str(user_id)

    # Сохраняем сессию и вопрос
    await repo.ensure_session(
        session_id=payload.session_id, 
        user_id=actual_user_id, 
        guest_id=actual_guest_id
    )
    
    await repo.save_message(
        session_id=payload.session_id,
        role="user",
        content=payload.question
    )
    
    initial_state = {
        "session_id": str(payload.session_id),
        "question": payload.question,
        
        # Строковые значения
        "current_query": "",
        "intent": "",
        "draft_answer": "",
        "critique": "",
        "final_answer": "",
        "current_step_message": "Инициализация...",
        "error": None,
        
        # Числовые счетчики и лимиты
        "search_count": 0,
        "revision_count": 0,
        "max_results": 3,
        
        # Булевые флаги
        "is_sufficient": False,
        "is_consistent": False,
        
        # Списки (Annotated с operator.add в LangGraph 
        # отлично работают, если на старте дать им пустой список)
        "internal_context": [],
        "web_context": []
    }

    config = {"configurable": {"thread_id": str(payload.session_id)}}

    langfuse_handler = get_langfuse_handler()
    if langfuse_handler:
        langfuse_handler.session_id = str(payload.session_id)
        # Langfuse принимает любые строки как user_id, даже "anonymous"
        langfuse_handler.user_id = str(user_id) if user_id else None
        config["callbacks"] = [langfuse_handler]

    async def event_generator():
        try:
            yield {"event": "status", "data": ChatEvent(node="start", message="init", model=model_name).model_dump_json()}

            # ❗️ Используем astream_events вместо astream
            async for event in graph.astream_events(initial_state, config=config, version="v2"):
                if await request.is_disconnected():
                    logger.info(f"[{payload.session_id}] Client disconnected, stopping stream.")
                    break

                kind = event["event"]

                if kind == "on_chat_model_stream" and "draft_generation" in event.get("tags", []):
                    chunk = event["data"]["chunk"].content
                    if chunk:
                        yield {"event": "token", "data": json.dumps({"text": chunk})}

                elif kind == "on_chain_end":
                    metadata = event.get("metadata", {})
                    if "langgraph_node" in metadata:
                        node_name = event["name"]
                        state_update = event["data"].get("output", {})

                        if isinstance(state_update, dict):
                            if "error" in state_update and state_update["error"]:
                                yield {"event": "error", "data": json.dumps({"detail": state_update["error"]})}
                                return
                            
                            if "current_step_message" in state_update:
                                step_msg = state_update.get("current_step_message", "")
                                chat_event = ChatEvent(node=node_name, message=step_msg, model=model_name)
                                yield {"event": "node_update", "data": chat_event.model_dump_json()}

            if not await request.is_disconnected():
                final_state = await graph.aget_state(config)
                final_text = final_state.values.get("final_answer", "Не удалось сформировать ответ.")
                
                trace_id = langfuse_handler.get_trace_id() if hasattr(langfuse_handler, 'get_trace_id') else None
                
                await repo.save_message(
                    session_id=payload.session_id,
                    role="assistant",
                    content=final_text,
                    trace_id=trace_id,
                    meta_data={"model": model_name}
                )

                final_answer = FinalAnswer(session_id=payload.session_id, answer=final_text, trace_id=trace_id)
                yield {"event": "final", "data": final_answer.model_dump_json()}

        except asyncio.CancelledError:
            logger.info(f"[{payload.session_id}] Stream cancelled by client.")
        except Exception as e:
            logger.exception(f"[{payload.session_id}] Stream error")
            yield {"event": "error", "data": json.dumps({"detail": "error"})} # Теперь отдаем код ошибки
        finally:
            if langfuse_handler and settings.langfuse_public_key:
                from langfuse import get_client
                get_client().flush()

    return EventSourceResponse(event_generator())