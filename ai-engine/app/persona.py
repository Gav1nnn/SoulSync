"""人设模块：结构化 PersonaProfile 与 Stage 1 mock 回复拼装。"""

from pydantic import BaseModel, Field


class PersonaProfile(BaseModel):
    """对应 V1 PersonaProfile，字段需结构化存储，避免整段 prompt 字符串。"""

    background: str = ""
    traits: list[str] = Field(default_factory=list)
    speaking_style: str = ""
    taboos: list[str] = Field(default_factory=list)
    expertise: list[str] = Field(default_factory=list)
    sample_lines: list[str] = Field(default_factory=list)


def default_berry_persona() -> PersonaProfile:
    """直连 /generate 且未传 persona 时的默认值；应与 Go DefaultBerryPersona 保持一致。"""
    return PersonaProfile(
        background="面向后端开发者的前端协作助手，擅长把需求拆成可落地的页面与接口方案。",
        traits=["务实", "直接", "有耐心", "偏工程化"],
        speaking_style="先澄清结构、状态和接口边界，再动手写前端；语气稳定，不堆术语。",
        taboos=["空泛鸡汤", "未验证就承诺效果", "跳过联调直接堆页面"],
        expertise=["Vue", "页面拆分", "组件设计", "接口联调", "前端 Debug"],
        sample_lines=["我是 Berry。", "先别急着乱堆页面。"],
    )


def build_mock_reply(persona: PersonaProfile, character_name: str, user_message: str) -> str:
    """用 persona 字段拼模板回复；改 persona 后文案会变，但不是 LLM 生成。"""
    opener = persona.sample_lines[0] if persona.sample_lines else f"我是 {character_name}。"
    style_hint = persona.speaking_style or "保持稳定、可执行的协作风格"
    expertise_hint = "、".join(persona.expertise[:3]) if persona.expertise else "前端协作"

    return (
        f"{opener}你这句「{user_message}」我已经接住了。"
        f"我会按「{style_hint}」来处理，并优先从 {expertise_hint} 角度帮你推进。"
        "先把结构、状态和接口边界捋顺，再开始写前端。"
    )
