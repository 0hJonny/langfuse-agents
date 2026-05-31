import logging
from fastapi import APIRouter, HTTPException
from api.schemas import FeedbackRequest
from core.config import settings
from langfuse import get_client
from core.postgres import db_pool                
from repositories.chat_repo import ChatRepository 

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/feedback", tags=["feedback"])

@router.post("/")
async def submit_feedback(feedback: FeedbackRequest):
    if not settings.langfuse_public_key or not settings.langfuse_secret_key:
        raise HTTPException(status_code=501, detail="LangFuse is not configured")

    try:
        client = get_client()
        trace_id = str(feedback.trace_id)

        # 1. Отправляем score в LangFuse
        client.create_score(
            trace_id=trace_id,
            name="user_feedback",
            value=1.0 if feedback.rating == "like" else 0.0,
            data_type="NUMERIC",
            comment=feedback.comment,
        )
        
        # 2. Сохраняем локально через репозиторий
        repo = ChatRepository(db_pool)
        saved = await repo.save_feedback_by_trace(
            trace_id=trace_id, 
            rating=feedback.rating, 
            comment=feedback.comment
        )
        
        if not saved:
            logger.warning(f"Feedback saved in Langfuse, but DB trace_id {trace_id} not found")

        logger.info(f"Feedback saved: session={trace_id} rating={feedback.rating}")
        return {"status": "ok", "message": "Feedback recorded"}
    except Exception as e:
        logger.error(f"Failed to save feedback: {e}")
        raise HTTPException(status_code=500, detail="Failed to store feedback")