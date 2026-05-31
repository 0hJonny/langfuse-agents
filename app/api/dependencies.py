from fastapi import Request, Header
from langgraph.graph.state import CompiledStateGraph

def get_graph(request: Request) -> CompiledStateGraph:
    """Извлекает скомпилированный граф из состояния приложения."""
    return request.app.state.graph

def get_current_user(x_user_id: str | None = Header(default=None, alias="X-User-Id")) -> str:
    """
    Извлекает ID пользователя из заголовка. 
    """
    return x_user_id or "anonymous"