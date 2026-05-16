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
    async def wrapper(state: AgentState) -> Dict[str, Any]:
        try:
            return await async_func(state)
        except Exception as e:
            logger.exception(f"Node {async_func.__name__} failed")
            return {
                "error": str(e),
                "current_step_message": f"Ошибка в узле {async_func.__name__}",
            }
    return wrapper

@safe_node
async def internal_search_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.session_id}] Local search")
    query = state.current_query or state.question
    results = await async_similarity_search(query, k=3)

    new_context = state.internal_context
    found = False
    for doc, distance in results:
        if distance < 1.5:
            found = True
            source = doc.metadata.get("source", "Локальная база")
            new_context += f"\n[Источник: {source}]: {doc.page_content}"

    msg = "Локальная база: данные найдены." if found else "Локальная база: ничего релевантного."
    return {"internal_context": new_context[:4000], "current_step_message": msg}

@safe_node
async def evaluate_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.session_id}] Evaluate")
    combined = f"ЛОКАЛЬНАЯ БАЗА:\n{state.internal_context}\n\nИНТЕРНЕТ:\n{state.web_context}"
    combined = truncate_text_by_tokens(combined, max_tokens=settings.max_context_tokens)

    if not combined.strip() or len(combined) < 50:
        return {"is_sufficient": False, "current_step_message": "Контекст пуст."}

    llm = get_llm()
    json_parser = JsonOutputParser(pydantic_object=EvaluationResult)
    prompt = ChatPromptTemplate.from_messages([
        ("system", "Ты — строгий контролер качества. {format_instructions}"),
        ("human", "Вопрос: {question}\n\nКонтекст:\n{context}")
    ])
    chain = prompt | llm | json_parser
    result = await safe_llm_ainvoke(chain, {
        "question": state.question,
        "context": combined,
        "format_instructions": json_parser.get_format_instructions(),
    })
    sufficient = result.get("is_sufficient", False)
    return {"is_sufficient": sufficient, "current_step_message": f"Достаточно? {'Да' if sufficient else 'Нет'}"}

@safe_node
async def reformulate_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.session_id}] Reformulate")
    llm = get_llm()
    current_date_str = datetime.now().strftime("%Y-%m-%d")
    instruction = "Придумай другой синоним." if state.web_context else "Сформулируй точный запрос."
    prompt = ChatPromptTemplate.from_messages([
        ("system", f"Ты — эксперт по поисковым запросам. Сегодня: {current_date_str}. Верни ТОЛЬКО запрос."),
        ("human", f"{instruction}\n\nВопрос: {{question}}")
    ])
    chain = prompt | llm | StrOutputParser()
    new_query = (await safe_llm_ainvoke(chain, {"question": state.question})).strip()
    new_max = state.max_results + 2 if state.web_context else state.max_results
    return {
        "current_query": new_query,
        "search_count": state.search_count + 1,
        "max_results": new_max,
        "current_step_message": f"Новый запрос: {new_query}",
    }

@safe_node
async def web_search_node(state: AgentState) -> Dict[str, Any]:
    search_target = state.current_query or state.question
    logger.info(f"[{state.session_id}] Web search: {search_target}")
    llm = get_llm()
    translate_prompt = ChatPromptTemplate.from_messages([
        ("system", "Translate to English. Return ONLY the translation."),
        ("human", "{query}")
    ])
    chain = translate_prompt | llm | StrOutputParser()
    english_query = (await safe_llm_ainvoke(chain, {"query": search_target})).strip()

    results = await safe_ddgs_search(english_query, state.max_results)
    if not results:
        return {"web_context": state.web_context, "current_step_message": "Нет результатов."}

    updated = state.web_context + f"\n--- Поиск по [{search_target}] ---\n"
    start = state.web_context.count("URL:") + 1
    for i, res in enumerate(results, start):
        url = res.get("href", res.get("link", res.get("url", "Без ссылки")))
        text = res.get("body", res.get("snippet", "Без текста"))
        updated += f"[{i}] URL: {url}\nТекст: {text}\n\n"
    return {"web_context": updated, "current_step_message": f"Найдено {len(results)} источников."}

@safe_node
async def synthesize_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.session_id}] Synthesize")
    llm = get_llm()
    context = f"<local>\n{state.internal_context}\n</local>\n<web>\n{state.web_context}\n</web>"
    context = truncate_text_by_tokens(context, max_tokens=settings.max_context_tokens)

    base_rules = "Пиши на русском. Факты со ссылками [1]. Раздел 'Источники:'."
    if state.critique:
        system = f"Исправь ответ. Замечания: {state.critique}\n{base_rules}"
    else:
        system = f"Ответь на вопрос строго по контексту.\n{base_rules}"

    prompt = ChatPromptTemplate.from_messages([
        ("system", system),
        ("human", "Контекст:\n{context}\n\nВопрос: {question}")
    ])
    chain = prompt | llm | StrOutputParser()
    draft = await safe_llm_ainvoke(chain, {"question": state.question, "context": context})
    return {"draft_answer": draft, "current_step_message": "Черновик готов."}

@safe_node
async def fact_check_node(state: AgentState) -> Dict[str, Any]:
    logger.info(f"[{state.session_id}] Fact check")
    llm = get_llm()
    fact_parser = JsonOutputParser(pydantic_object=FactCheckResult)
    context = f"LOCAL:\n{state.internal_context}\nWEB:\n{state.web_context}"
    context = truncate_text_by_tokens(context, max_tokens=settings.max_context_tokens)

    prompt = ChatPromptTemplate.from_messages([
        ("system", "Ты — Фактчекер. Проверь черновик. {format_instructions}"),
        ("human", "Вопрос: {question}\nКонтекст:\n{context}\nЧерновик:\n{draft}")
    ])
    chain = prompt | llm | fact_parser
    result = await safe_llm_ainvoke(chain, {
        "question": state.question,
        "context": context,
        "draft": state.draft_answer,
        "format_instructions": fact_parser.get_format_instructions(),
    })
    consistent = result.get("is_consistent", True)
    critique = result.get("reasoning", "")
    return {
        "is_consistent": consistent,
        "critique": critique,
        "revision_count": state.revision_count + 1,
        "current_step_message": f"Фактчекинг: {'пройден' if consistent else 'найдены ошибки'}.",
    }

@safe_node
async def finalize_node(state: AgentState) -> Dict[str, Any]:
    final = state.draft_answer
    if not state.is_consistent:
        final = "⚠️ *Ответ может содержать неточности.*\n\n" + final
    if not state.is_sufficient:
        final = "ℹ️ *Полного ответа не найдено.*\n\n" + final
    return {"final_answer": final, "current_step_message": "Ответ готов."}