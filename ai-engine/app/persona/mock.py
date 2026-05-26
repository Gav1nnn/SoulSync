from app.persona.profile import PersonaProfile


def build_mock_reply(persona: PersonaProfile, character_name: str, user_message: str) -> str:
    opener = persona.sample_lines[0] if persona.sample_lines else f"我是 {character_name}。"
    style_hint = persona.speaking_style or "保持稳定、可执行的协作风格"
    expertise_hint = "、".join(persona.expertise[:3]) if persona.expertise else "前端协作"

    return (
        f"{opener}你这句「{user_message}」我已经接住了。"
        f"我会按「{style_hint}」来处理，并优先从 {expertise_hint} 角度帮你推进。"
        "先把结构、状态和接口边界捋顺，再开始写前端。"
    )
