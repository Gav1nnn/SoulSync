"""编排层：串联 persona 注入 → 生成（LLM 或 mock）→ 评估。"""

from core.config import _llm_enabled, _llm_provider, _llm_mode
from core.evaluation import build_generate_response
from core.persona import build_mock_reply, generate_with_persona
from core.schemas import GenerateRequest, GenerateResponse


def generate_reply(request: GenerateRequest, rag) -> GenerateResponse:
    """LLM 已配置时走 DeepSeek 等；失败或未配置则 mock。"""
    user_message = request.user_message.strip()
    llm_provider = _llm_provider()
    llm_mode = _llm_mode()

    if _llm_enabled():
        print("调用API")
        if llm_mode == "persona":
            print("性格回复")
            reply = generate_with_persona(
                request.persona,
                request.character_name,
                user_message,
                rag
            )
        elif llm_mode == "rag":
            print("rag回复")
            reply = rag._ask_llm_rag(user_message)
        else:
            print("默认回复1")
            reply = build_mock_reply(
                request.persona,
                request.character_name,
                user_message,
            )
    else:
        print("默认回复2")
        reply = build_mock_reply(
            request.persona,
            request.character_name,
            user_message,
        )

    return build_generate_response(
        reply=reply,
        character_name=request.character_name,
        persona=request.persona,
        llm_provider=llm_provider,
    )
