from app.llm.client import LLMClientError, extract_memory_candidates, generate_persona_reply_with_context
from app.persona.mock import build_mock_reply
from app.retrieval.retriever import retrieve_knowledge_result
from app.schemas import GenerateRequest, GenerateResponse


def generate_reply(request: GenerateRequest) -> GenerateResponse:
    context_used = ["persona"]
    context_used.append("persona.examples")
    used_memory_ids = [memory.id for memory in request.memories]
    if used_memory_ids:
        context_used.append("memory")
    knowledge_hits = []
    retrieval_strategies: list[str] = []

    try:
        retrieval_result = retrieve_knowledge_result(request.user_message)
        knowledge_hits = retrieval_result.hits
        retrieval_strategies = retrieval_result.strategies
    except Exception:
        context_used.append("knowledge_unavailable")

    if knowledge_hits:
        context_used.append("knowledge")
        context_used.extend(retrieval_strategies)
        used_knowledge_chunk_ids = [hit.chunk.chunk_id for hit in knowledge_hits]
    else:
        context_used.extend(retrieval_strategies)
        used_knowledge_chunk_ids = []

    generated_with_llm = False
    try:
        reply, provider = generate_persona_reply_with_context(
            request.persona,
            request.character_name,
            request.user_message,
            knowledge_hits,
            request.memories,
        )
        generated_with_llm = True
        context_used.append(provider)
    except LLMClientError:
        reply = build_mock_reply(
            request.persona,
            request.character_name,
            request.user_message,
            knowledge_hits,
        )
        context_used.append("mock_fallback")

    memory_candidates = []
    if generated_with_llm:
        try:
            memory_candidates = extract_memory_candidates(request.user_message, reply)
        except LLMClientError:
            memory_candidates = []

    return GenerateResponse(
        reply=reply,
        persona=request.character_name,
        context_used=context_used,
        used_persona=True,
        used_memory_ids=used_memory_ids,
        used_knowledge_chunk_ids=used_knowledge_chunk_ids,
        memory_candidates=memory_candidates,
    )
