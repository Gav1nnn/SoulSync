from app.llm.client import LLMClientError, generate_persona_reply_with_knowledge
from app.persona.mock import build_mock_reply
from app.retrieval.retriever import retrieve_knowledge_result
from app.schemas import GenerateRequest, GenerateResponse


def generate_reply(request: GenerateRequest) -> GenerateResponse:
    context_used = ["persona"]
    context_used.append("persona.examples")
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

    try:
        reply, provider = generate_persona_reply_with_knowledge(
            request.persona,
            request.character_name,
            request.user_message,
            knowledge_hits,
        )
        context_used.append(provider)
    except LLMClientError:
        reply = build_mock_reply(
            request.persona,
            request.character_name,
            request.user_message,
            knowledge_hits,
        )
        context_used.append("mock_fallback")

    return GenerateResponse(
        reply=reply,
        persona=request.character_name,
        context_used=context_used,
        used_persona=True,
        used_knowledge_chunk_ids=used_knowledge_chunk_ids,
    )
