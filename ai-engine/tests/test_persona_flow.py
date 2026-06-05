import unittest
from pathlib import Path
from unittest.mock import patch

from app.llm.client import LLMClientError, current_provider, load_settings
from app.orchestration.generate_reply import generate_reply
from app.persona.examples import berry_few_shot_messages
from app.persona.profile import PersonaProfile, default_berry_persona
from app.persona.prompt_builder import build_persona_instruction
from app.retrieval.config import RetrievalSettings
from app.retrieval.chunker import chunk_markdown_document
from app.retrieval.embedder import EmbeddingUnavailableError
from app.retrieval.retriever import retrieve_knowledge_result, tokenize
from app.retrieval.schemas import RetrievalHit, RetrievalResult
from app.schemas import GenerateRequest, MemoryCandidate, MemoryContext


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
        self.assertIn("毒舌学姐感", instruction)
        self.assertIn("不要空泛鸡汤", instruction)

    def test_few_shot_messages_are_available(self) -> None:
        messages = berry_few_shot_messages()

        self.assertGreaterEqual(len(messages), 8)
        self.assertEqual(messages[0]["role"], "user")
        self.assertEqual(messages[1]["role"], "assistant")
        self.assertIn("想得倒挺省事", messages[1]["content"])

    @patch("app.orchestration.generate_reply.extract_memory_candidates")
    @patch("app.orchestration.generate_reply.retrieve_knowledge_result")
    @patch("app.orchestration.generate_reply.generate_persona_reply_with_context")
    def test_generate_reply_uses_provider_context_on_success(
        self,
        mock_generate_persona_reply_with_context,
        mock_retrieve_knowledge_result,
        mock_extract_memory_candidates,
    ) -> None:
        chunk = chunk_markdown_document(
            "docs/architecture.md",
            "# Current\n用户输入 -> frontend -> backend",
        )[0]
        mock_retrieve_knowledge_result.return_value = RetrievalResult(
            hits=[RetrievalHit(chunk=chunk, score=0.92)],
            strategies=["knowledge.embedding", "knowledge.keyword"],
        )
        mock_generate_persona_reply_with_context.return_value = ("Berry real reply", "ollama")
        mock_extract_memory_candidates.return_value = [
            MemoryCandidate(
                type="project_fact",
                content="项目通过 Go 调用 ai-engine。",
                reason="用户询问当前项目的人设和链路。",
                confidence=0.82,
            )
        ]
        request = GenerateRequest(user_message="你记得你的人设是什么吗")

        response = generate_reply(request)

        self.assertEqual(response.reply, "Berry real reply")
        self.assertEqual(response.persona, "Berry")
        self.assertEqual(
            response.context_used,
            ["persona", "persona.examples", "knowledge", "knowledge.embedding", "knowledge.keyword", "ollama"],
        )
        self.assertTrue(response.used_persona)
        self.assertEqual(response.used_memory_ids, [])
        self.assertEqual(response.used_knowledge_chunk_ids, [chunk.chunk_id])
        self.assertFalse(response.memory_written)
        self.assertEqual(len(response.memory_candidates), 1)

    @patch("app.orchestration.generate_reply.extract_memory_candidates")
    @patch("app.orchestration.generate_reply.retrieve_knowledge_result")
    @patch("app.orchestration.generate_reply.generate_persona_reply_with_context")
    def test_generate_reply_falls_back_to_mock_when_llm_fails(
        self,
        mock_generate_persona_reply_with_context,
        mock_retrieve_knowledge_result,
        mock_extract_memory_candidates,
    ) -> None:
        mock_retrieve_knowledge_result.return_value = RetrievalResult(hits=[], strategies=[])
        mock_generate_persona_reply_with_context.side_effect = LLMClientError("llm request timed out")
        request = GenerateRequest(user_message="帮我拆一个列表页")

        response = generate_reply(request)

        self.assertIn("我是 Berry。", response.reply)
        self.assertEqual(response.context_used, ["persona", "persona.examples", "mock_fallback"])
        self.assertTrue(response.used_persona)
        self.assertEqual(response.used_knowledge_chunk_ids, [])
        mock_extract_memory_candidates.assert_not_called()

    @patch("app.orchestration.generate_reply.extract_memory_candidates")
    @patch("app.orchestration.generate_reply.retrieve_knowledge_result")
    @patch("app.orchestration.generate_reply.generate_persona_reply_with_context")
    def test_generate_reply_injects_memories(
        self,
        mock_generate_persona_reply_with_context,
        mock_retrieve_knowledge_result,
        mock_extract_memory_candidates,
    ) -> None:
        mock_retrieve_knowledge_result.return_value = RetrievalResult(hits=[], strategies=[])
        mock_generate_persona_reply_with_context.return_value = ("Berry memory reply", "ollama")
        mock_extract_memory_candidates.return_value = []
        request = GenerateRequest(
            user_message="按我们的项目约定继续拆页面",
            memories=[
                MemoryContext(
                    id="mem-1",
                    type="frontend_convention",
                    content="项目页面优先按容器组件和业务组件拆分。",
                )
            ],
        )

        response = generate_reply(request)

        self.assertEqual(response.used_memory_ids, ["mem-1"])
        self.assertEqual(response.context_used, ["persona", "persona.examples", "memory", "ollama"])
        mock_generate_persona_reply_with_context.assert_called_once()

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
            "DEEPSEEK_MODEL": "deepseek-v4-flash",
        },
        clear=True,
    )
    def test_load_settings_disables_deepseek_without_api_key(self) -> None:
        settings = load_settings()

        self.assertEqual(settings.provider, "deepseek")
        self.assertFalse(settings.enabled)

    @patch.dict(
        "os.environ",
        {
            "LLM_DISABLED": "false",
            "LLM_PROVIDER": "deepseek",
            "DEEPSEEK_API_KEY": "test-key",
            "DEEPSEEK_BASE_URL": "https://api.deepseek.com",
        },
        clear=True,
    )
    def test_load_settings_uses_current_deepseek_defaults(self) -> None:
        settings = load_settings()

        self.assertTrue(settings.enabled)
        self.assertEqual(settings.provider, "deepseek")
        self.assertEqual(settings.model, "deepseek-v4-flash")
        self.assertTrue(settings.thinking_disabled)

    @patch.dict(
        "os.environ",
        {
            "LLM_DISABLED": "false",
            "LLM_PROVIDER": "deepseek",
            "DEEPSEEK_API_KEY": "test-key",
            "DEEPSEEK_THINKING_ENABLED": "true",
        },
        clear=True,
    )
    def test_load_settings_can_enable_deepseek_thinking(self) -> None:
        settings = load_settings()

        self.assertFalse(settings.thinking_disabled)

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

    def test_tokenize_handles_chinese_and_paths(self) -> None:
        tokens = tokenize("POST /generate 对应 docs/architecture.md")

        self.assertIn("post", tokens)
        self.assertIn("/generate", tokens)
        self.assertIn("docs/architecture.md", tokens)

    def test_chunk_markdown_document_splits_by_heading(self) -> None:
        chunks = chunk_markdown_document(
            "docs/test.md",
            "# Intro\n第一段\n\n## API\nPOST /generate\n\n第二段",
        )

        self.assertEqual(len(chunks), 3)
        self.assertEqual(chunks[0].section, "Intro")
        self.assertEqual(chunks[1].section, "API")
        self.assertEqual(chunks[2].section, "API")

    @patch("app.retrieval.retriever.embedding_hits_for_query")
    @patch("app.retrieval.retriever.load_retrieval_settings")
    @patch("app.retrieval.retriever.cached_chunks")
    def test_retrieve_knowledge_result_fuses_embedding_and_keyword_hits(
        self,
        mock_cached_chunks,
        mock_load_retrieval_settings,
        mock_embedding_hits_for_query,
    ) -> None:
        chunks = tuple(
            chunk_markdown_document(
                "docs/architecture.md",
                "# Generate\nPOST /generate 会调用 ai-engine\n\n# Health\nGET /healthz",
            )
        )
        mock_cached_chunks.return_value = chunks
        mock_load_retrieval_settings.return_value = RetrievalSettings(
            enabled=True,
            provider="ollama",
            base_url="http://127.0.0.1:11434",
            model_name="qwen3-embedding:0.6b",
            cache_dir=Path("/tmp/rag-test-cache"),
            top_k=3,
            keyword_top_k=6,
            embedding_min_score=0.15,
            timeout_seconds=60.0,
            query_prefix="query:",
            document_prefix="doc:",
        )
        mock_embedding_hits_for_query.return_value = [RetrievalHit(chunk=chunks[0], score=0.88)]

        result = retrieve_knowledge_result("POST /generate persona")

        self.assertGreaterEqual(len(result.hits), 1)
        self.assertEqual(result.hits[0].chunk.chunk_id, chunks[0].chunk_id)
        self.assertEqual(result.strategies, ["knowledge.embedding", "knowledge.keyword"])

    @patch("app.retrieval.retriever.embedding_hits_for_query")
    @patch("app.retrieval.retriever.load_retrieval_settings")
    @patch("app.retrieval.retriever.cached_chunks")
    def test_retrieve_knowledge_result_falls_back_to_keyword_hits(
        self,
        mock_cached_chunks,
        mock_load_retrieval_settings,
        mock_embedding_hits_for_query,
    ) -> None:
        chunks = tuple(
            chunk_markdown_document(
                "docs/architecture.md",
                "# Generate\nPOST /generate 会调用 ai-engine",
            )
        )
        mock_cached_chunks.return_value = chunks
        mock_load_retrieval_settings.return_value = RetrievalSettings(
            enabled=True,
            provider="ollama",
            base_url="http://127.0.0.1:11434",
            model_name="qwen3-embedding:0.6b",
            cache_dir=Path("/tmp/rag-test-cache"),
            top_k=3,
            keyword_top_k=6,
            embedding_min_score=0.15,
            timeout_seconds=60.0,
            query_prefix="query:",
            document_prefix="doc:",
        )
        mock_embedding_hits_for_query.side_effect = EmbeddingUnavailableError("model missing")

        result = retrieve_knowledge_result("POST /generate persona")

        self.assertGreaterEqual(len(result.hits), 1)
        self.assertEqual(result.strategies, ["knowledge.keyword", "knowledge.lexical_fallback"])
        self.assertTrue(all(hit.chunk.chunk_id for hit in result.hits))


if __name__ == "__main__":
    unittest.main()
