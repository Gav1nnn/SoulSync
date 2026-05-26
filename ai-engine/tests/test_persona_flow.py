import unittest
from unittest.mock import patch

from app.llm.client import LLMClientError, current_provider, load_settings
from app.orchestration.generate_reply import generate_reply
from app.persona.examples import berry_few_shot_messages
from app.persona.profile import PersonaProfile, default_berry_persona
from app.persona.prompt_builder import build_persona_instruction
from app.schemas import GenerateRequest


class PersonaFlowTests(unittest.TestCase):
    def test_generate_request_uses_default_berry_persona(self) -> None:
        request = GenerateRequest(user_message="帮我拆一个列表页")

        self.assertEqual(request.character_id, "berry")
        self.assertEqual(request.character_name, "Berry")
        self.assertEqual(request.persona, default_berry_persona())

    def test_prompt_builder_contains_core_persona_fields(self) -> None:
        persona = default_berry_persona()

        instruction = build_persona_instruction(persona, "Berry")

        self.assertIn("Berry", instruction)
        self.assertIn(persona.background, instruction)
        self.assertIn("Vue", instruction)
        self.assertIn("不要空泛鸡汤", instruction)

    def test_few_shot_messages_are_available(self) -> None:
        messages = berry_few_shot_messages()

        self.assertGreaterEqual(len(messages), 8)
        self.assertEqual(messages[0]["role"], "user")
        self.assertEqual(messages[1]["role"], "assistant")

    @patch("app.orchestration.generate_reply.generate_persona_reply")
    def test_generate_reply_uses_provider_context_on_success(self, mock_generate_persona_reply) -> None:
        mock_generate_persona_reply.return_value = ("Berry real reply", "ollama")
        request = GenerateRequest(user_message="你记得你的人设是什么吗")

        response = generate_reply(request)

        self.assertEqual(response.reply, "Berry real reply")
        self.assertEqual(response.persona, "Berry")
        self.assertEqual(response.context_used, ["persona", "persona.examples", "ollama"])
        self.assertTrue(response.used_persona)
        self.assertEqual(response.used_memory_ids, [])
        self.assertEqual(response.used_knowledge_chunk_ids, [])
        self.assertFalse(response.memory_written)

    @patch("app.orchestration.generate_reply.generate_persona_reply")
    def test_generate_reply_falls_back_to_mock_when_llm_fails(self, mock_generate_persona_reply) -> None:
        mock_generate_persona_reply.side_effect = LLMClientError("llm request timed out")
        request = GenerateRequest(user_message="帮我拆一个列表页")

        response = generate_reply(request)

        self.assertIn("我是 Berry。", response.reply)
        self.assertEqual(response.context_used, ["persona", "persona.examples", "mock_fallback"])
        self.assertTrue(response.used_persona)

    @patch.dict(
        "os.environ",
        {
            "LLM_DISABLED": "false",
            "LLM_PROVIDER": "ollama",
            "LLM_BASE_URL": "http://127.0.0.1:11434/v1",
            "LLM_MODEL": "qwen3.5:4b",
            "LLM_TIMEOUT_SECONDS": "180",
        },
        clear=True,
    )
    def test_load_settings_prefers_ollama_defaults(self) -> None:
        settings = load_settings()

        self.assertEqual(current_provider(), "ollama")
        self.assertTrue(settings.enabled)
        self.assertEqual(settings.provider, "ollama")
        self.assertEqual(settings.model, "qwen3.5:4b")
        self.assertEqual(settings.timeout_seconds, 180.0)

    @patch.dict(
        "os.environ",
        {
            "LLM_DISABLED": "false",
            "LLM_PROVIDER": "deepseek",
            "DEEPSEEK_API_KEY": "",
            "DEEPSEEK_BASE_URL": "https://api.deepseek.com",
            "DEEPSEEK_MODEL": "deepseek-chat",
        },
        clear=True,
    )
    def test_load_settings_disables_deepseek_without_api_key(self) -> None:
        settings = load_settings()

        self.assertEqual(settings.provider, "deepseek")
        self.assertFalse(settings.enabled)

    def test_prompt_builder_works_with_custom_persona(self) -> None:
        persona = PersonaProfile(
            background="毒舌但靠谱的二次元前端学姐搭子。",
            traits=["毒舌", "靠谱"],
            speaking_style="先吐槽，再给能落地的方案。",
            taboos=["假大空"],
            expertise=["Vue", "Debug"],
            sample_lines=["你先别急。"],
        )

        instruction = build_persona_instruction(persona, "Berry")

        self.assertIn("毒舌但靠谱的二次元前端学姐搭子。", instruction)
        self.assertIn("先吐槽，再给能落地的方案。", instruction)
        self.assertIn("你先别急。", instruction)


if __name__ == "__main__":
    unittest.main()
