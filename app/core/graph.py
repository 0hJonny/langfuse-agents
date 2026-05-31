import logging
from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from agent.workflow import build_agent_workflow

logger = logging.getLogger(__name__)

async def init_agent_app(checkpointer: AsyncPostgresSaver):
    """
    Инфраструктурная функция: берет бизнес-логику агента и оборачивает 
    ее в механизмы сохранения состояния (checkpointer) для FastAPI.
    """
    logger.info("Инициализация и компиляция LangGraph агента...")
    
    # Получаем чистый граф
    workflow = build_agent_workflow()
    
    # Компилируем его с чекпоинтером
    app = workflow.compile(checkpointer=checkpointer)
    
    logger.info("Агент успешно скомпилирован и готов к работе.")
    return app