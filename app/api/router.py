from fastapi import APIRouter
from api import chat, feedback

api_v1_router = APIRouter(prefix="/api/v1")

api_v1_router.include_router(chat.router)
api_v1_router.include_router(feedback.router)