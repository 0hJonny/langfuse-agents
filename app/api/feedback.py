import logging
from fastapi import APIRouter, HTTPException
from api.schemas import FeedbackRequest
from core.config import settings
from langfuse import get_client

logger = logging.getLogger(__name__)
router = APIRouter(tags=["feedback"])

@router.post("/chat/feedback")
async def submit_feedback(feedback: FeedbackRequest):
    if not settings.langfuse_public_key or not settings.langfuse_secret_key:
        raise HTTPException(status_code=501, detail="LangFuse is not configured")

    try:
        client = get_client()
        # Используем session_id как идентификатор трейса (он же thread_id в чекпоинтах)
        trace_id = str(feedback.session_id)

        # Отправляем score в LangFuse
        client.create_score(
            trace_id=trace_id,
            name="user_feedback",
            value=1.0 if feedback.rating == "like" else 0.0,
            data_type="BOOLEAN",
            comment=feedback.comment,
        )
        logger.info(f"Feedback saved: session={trace_id} rating={feedback.rating}")
        return {"status": "ok", "message": "Feedback recorded"}
    except Exception as e:
        logger.error(f"Failed to save feedback: {e}")
        raise HTTPException(status_code=500, detail="Failed to store feedback")