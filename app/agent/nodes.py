import asyncio
import logging
from datetime import datetime
from typing import Any, Dict
from ddgs import DDGS
from ddgs.exceptions import DDGSException
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.output_parsers import StrOutputParser, JsonOutputParser
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type

from core.llm_provider import get_llm
from core.config import settings
from storage.chroma_client import async_similarity_search
from agent.state import AgentState
from shared_types.models import EvaluationResult, FactCheckResult
from shared_types.constants import StepCode
from utils.token_counter import truncate_text_by_tokens

logger = logging.getLogger(__name__)

@retry(stop=stop_after_attempt(3), wait=wait_exponential(multiplier=1, min=2, max=10))
async def safe_llm_ainvoke(chain, payload):
    return await chain.ainvoke(payload)

@retry(
    stop=stop_after_attempt(2),
    wait=wait_exponential(multiplier=1, min=1, max=5),
    retry=retry_if_exception_type((ConnectionError, TimeoutError))
)
async def _ddgs_search(query: str, max_results: int):
    """Непосредственный вызов DDGS (с retry для сетевых ошибок)."""
    return await asyncio.to_thread(
        lambda: list(DDGS().text(query, max_results=max_results, backend="auto"))
    )

async def safe_ddgs_search(query: str, max_results: int):
    """Обёртка: при любой ошибке возвращает пустой список."""
    try:
        return await _ddgs_search(query, max_results)
    except DDGSException as e:
        logger.warning(f"DuckDuckGo search returned no results: {e}")
        return []
    except Exception as e:
        logger.error(f"DuckDuckGo search unexpected error: {e}")
        return []

def safe_node(async_func):
    """Декоратор для перехвата исключений на уровне узла графа."""
    async def wrapper(state: AgentState) -> Dict[str, Any]:
        try:
            return await async_func(state)
        except Exception as e:
            logger.exception(f"Node {async_func.__name__} failed")
            return {
                "error": str(e),
                "current_step_message": StepCode.ERROR.value,
            }
    return wrapper

@safe_node
async def classify_intent_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.get('session_id')}] Классификация запроса")
    llm = get_llm()
    
    prompt = ChatPromptTemplate.from_messages([
        ("system", """Определи тип вопроса пользователя. 
        Верни ТОЛЬКО одно из слов:
        - 'chitchat' (приветствия, базовые диалоги)
        - 'capabilities' (вопросы о том, что ты умеешь)
        - 'rag' (любые предметные вопросы, требующие поиска фактов)"""),
        ("human", "{question}")
    ])
    
    chain = prompt | llm | StrOutputParser()
    intent = (await safe_llm_ainvoke(chain, {"question": state.get("question")})).strip().lower()
    
    if intent not in ["chitchat", "capabilities", "rag"]:
        intent = "rag"
        
    return {
        "intent": intent, 
        "current_step_message": f"intent_{intent}"
    }

@safe_node
async def simple_response_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.get('session_id')}] Простой ответ")
    llm = get_llm()
    
    system_msg = "Ты — умный AI-ассистент."
    if state.get("intent") == "capabilities":
        system_msg += " Расскажи кратко, что ты умеешь искать информацию в локальной базе и интернете, проводить фактчекинг и давать точные ответы."
        
    prompt = ChatPromptTemplate.from_messages([
        ("system", system_msg),
        ("human", "{question}")
    ])
    
    chain = prompt | llm | StrOutputParser()
    response = await safe_llm_ainvoke(chain, {"question": state.get("question")})
    
    return {
        "final_answer": response, 
        "current_step_message": StepCode.FINALIZING.value
    }

@safe_node
async def internal_search_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.get('session_id')}] Local search")
    query = state.get("current_query") or state.get("question")
    results = await async_similarity_search(query, k=3)

    new_findings = []
    for doc, distance in results:
        if distance < 1.5:
            source = doc.metadata.get("source", "Локальная база")
            new_findings.append(f"[Источник: {source}]: {doc.page_content}")

    msg_code = StepCode.LOCAL_FOUND.value if new_findings else StepCode.LOCAL_NOT_FOUND.value
    
    return {
        "internal_context": new_findings, 
        "current_step_message": msg_code
    }

@safe_node
async def evaluate_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.get('session_id')}] Evaluate")
    
    internal_str = "\n".join(state.get("internal_context", []))
    web_str = "\n".join(state.get("web_context", []))
    
    combined = f"ЛОКАЛЬНАЯ БАЗА:\n{internal_str}\n\nИНТЕРНЕТ:\n{web_str}"
    combined = truncate_text_by_tokens(combined, max_tokens=settings.max_context_tokens)

    if not combined.strip() or len(combined) < 50:
        return {
            "is_sufficient": False, 
            "current_step_message": StepCode.EVALUATING.value
        }

    llm = get_llm()
    json_parser = JsonOutputParser(pydantic_object=EvaluationResult)
    prompt = ChatPromptTemplate.from_messages([
        ("system", "Ты — строгий контролер качества. {format_instructions}"),
        ("human", "Вопрос: {question}\n\nКонтекст:\n{context}")
    ])
    chain = prompt | llm | json_parser
    result = await safe_llm_ainvoke(chain, {
        "question": state.get("question"),
        "context": combined,
        "format_instructions": json_parser.get_format_instructions(),
    })
    sufficient = result.get("is_sufficient", False)
    
    return {
        "is_sufficient": sufficient, 
        "current_step_message": StepCode.EVALUATING.value
    }

@safe_node
async def reformulate_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.get('session_id')}] Reformulate")
    llm = get_llm()
    current_date_str = datetime.now().strftime("%Y-%m-%d")
    instruction = "Придумай другой синоним." if state.get("web_context") else "Сформулируй точный запрос."
    
    prompt = ChatPromptTemplate.from_messages([
        ("system", f"Ты — эксперт по поисковым запросам. Сегодня: {current_date_str}. Верни ТОЛЬКО запрос."),
        ("human", f"{instruction}\n\nВопрос: {{question}}")
    ])
    chain = prompt | llm | StrOutputParser()
    new_query = (await safe_llm_ainvoke(chain, {"question": state.get("question")})).strip()
    
    new_max = state.get("max_results", 3) + 2 if state.get("web_context") else state.get("max_results", 3)
    current_search_count = state.get("search_count", 0)
    
    return {
        "current_query": new_query,
        "search_count": current_search_count + 1,
        "max_results": new_max,
        "current_step_message": StepCode.REFORMULATING.value,
    }

@safe_node
async def web_search_node(state: AgentState) -> Dict[str, Any]:
    search_target = state.get("current_query") or state.get("question")
    logger.info(f"[{state.get('session_id')}] Web search: {search_target}")
    llm = get_llm()
    translate_prompt = ChatPromptTemplate.from_messages([
        ("system", "Translate to English. Return ONLY the translation."),
        ("human", "{query}")
    ])
    chain = translate_prompt | llm | StrOutputParser()
    english_query = (await safe_llm_ainvoke(chain, {"query": search_target})).strip()

    results = await safe_ddgs_search(english_query, state.get("max_results", 3))
    if not results:
        return {"current_step_message": StepCode.WEB_NOT_FOUND.value}

    search_result_text = f"\n--- Поиск по [{search_target}] ---\n"
    start = len(state.get("web_context", [])) + 1 
    
    for i, res in enumerate(results, 1):
        url = res.get("href", res.get("link", res.get("url", "Без ссылки")))
        text = res.get("body", res.get("snippet", "Без текста"))
        search_result_text += f"[{start}.{i}] URL: {url}\nТекст: {text}\n\n"
        
    return {
        "web_context": [search_result_text], 
        "current_step_message": StepCode.WEB_FOUND.value
    }

@safe_node
async def synthesize_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.get('session_id')}] Synthesize")
    llm = get_llm()
    
    internal_str = "\n".join(state.get("internal_context", []))
    web_str = "\n".join(state.get("web_context", []))
    context = f"<local>\n{internal_str}\n</local>\n<web>\n{web_str}\n</web>"
    context = truncate_text_by_tokens(context, max_tokens=settings.max_context_tokens)

    base_rules = "Пиши на русском. Факты со ссылками [1]. Раздел 'Источники:'."
    if state.get("critique"):
        system = f"Исправь ответ. Замечания: {state.get('critique')}\n{base_rules}"
    else:
        system = f"Ответь на вопрос строго по контексту.\n{base_rules}"

    prompt = ChatPromptTemplate.from_messages([
        ("system", system),
        ("human", "Контекст:\n{context}\n\nВопрос: {question}")
    ])
    
    chain = (prompt | llm | StrOutputParser()).with_config({"tags": ["draft_generation"]})
    
    draft = await safe_llm_ainvoke(chain, {"question": state.get("question"), "context": context})
    
    return {
        "draft_answer": draft, 
        "current_step_message": StepCode.SYNTHESIZING.value
    }

@safe_node
async def fact_check_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.get('session_id')}] Fact check")
    llm = get_llm()
    fact_parser = JsonOutputParser(pydantic_object=FactCheckResult)
    
    internal_str = "\n".join(state.get("internal_context", []))
    web_str = "\n".join(state.get("web_context", []))
    context = f"LOCAL:\n{internal_str}\nWEB:\n{web_str}"
    context = truncate_text_by_tokens(context, max_tokens=settings.max_context_tokens)

    prompt = ChatPromptTemplate.from_messages([
        ("system", "Ты — Фактчекер. Проверь черновик. {format_instructions}"),
        ("human", "Вопрос: {question}\nКонтекст:\n{context}\nЧерновик:\n{draft}")
    ])
    chain = prompt | llm | fact_parser
    result = await safe_llm_ainvoke(chain, {
        "question": state.get("question"),
        "context": context,
        "draft": state.get("draft_answer"),
        "format_instructions": fact_parser.get_format_instructions(),
    })
    
    consistent = result.get("is_consistent", True)
    critique = result.get("reasoning", "")
    current_revision = state.get("revision_count", 0)
    
    return {
        "is_consistent": consistent,
        "critique": critique,
        "revision_count": current_revision + 1,
        "current_step_message": StepCode.FACT_CHECKING.value,
    }

@safe_node
async def finalize_node(state: AgentState) -> Dict[str, Any]:
    final = state.get("draft_answer", "")
    
    if not state.get("is_consistent", True):
        final = "⚠️ *Ответ может содержать неточности.*\n\n" + final
    if not state.get("is_sufficient", True):
        final = "ℹ️ *Полного ответа не найдено.*\n\n" + final
        
    return {
        "final_answer": final, 
        "current_step_message": StepCode.FINALIZING.value
    }