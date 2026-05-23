# 使用 LangChain 加载目录中所有文档

"""
# 下载 NLTK 所需的数据包。需要在你的 Python 环境中运行以下代码一次
import nltk
nltk.download('averaged_perceptron_tagger')
nltk.download('punkt')
"""
import os
from langchain_community.document_loaders import DirectoryLoader
from sentence_transformers import SentenceTransformer
import faiss  # pip install faiss-cpu
import numpy as np
from openai import OpenAI
from dotenv import load_dotenv
load_dotenv()
from core.config import _env

class RAG:
    def __init__(self):
        self.docs = None
        self.doc_embeddings = None
        self.embedding_model = None
        self.index = None
        self.client = None
        self.chat_model = _env("LLM_MODEL")

    def _run_rag(self):
        print("RAG初始化中")
        self._load_docs()
        self._embedding()
        self._create_vector()
        self.client = OpenAI(
            api_key=_env("LLM_API_KEY"),
            base_url=_env("LLM_BASE_URL")
        )

    def _load_docs(self, path = '../../资料'):
        # 获取当前脚本文件所在的目录
        print("读取文件中……")
        script_dir = os.path.dirname(__file__)
        #print(f"获取当前脚本文件所在的目录：{script_dir}")
        data_dir = os.path.join(script_dir, path)
        loader = DirectoryLoader(data_dir)
        self.docs = loader.load()
        print(f"读取完成。文档数：{len(self.docs)}")  # 输出文档总数
        #print(docs[0])  # 输出第一个文档

    def _embedding(self, path = "../../embedding_model/Qwen3-Embedding-0___6B"):
        print("文本嵌入中……")
        script_dir = os.path.dirname(__file__)
        model_path = os.path.join(script_dir, path)
        self.embedding_model = SentenceTransformer(model_path)
        texts = [doc.page_content for doc in self.docs]
        self.doc_embeddings = self.embedding_model.encode(texts)
        print(f"文本嵌入完成。文档向量维度: {self.doc_embeddings.shape}")

    def _create_vector(self):
        print("建立向量数据库中……")
        dimension = self.doc_embeddings.shape[1]
        self.index = faiss.IndexFlatL2(dimension)
        self.index.add(self.doc_embeddings.astype('float32'))
        print(f"数据库建立完成。向量数据库中的文档数量: {self.index.ntotal}")

    def _search_vector(self,question):
        print("检索向量数据库中")
        query_embedding = self.embedding_model.encode([question])[0]
        distances, indices = self.index.search(
            np.array([query_embedding]).astype('float32'),
            k=3
        )
        context = [self.docs[idx] for idx in indices[0]]
        print("\n检索到的相关文档:")
        for i, doc in enumerate(context, 1):
            print(f"[{i}] {doc}")
        return context

    def _ask_llm_rag(self, question):
        context = self._search_vector(question)
        prompt = f"""根据以下参考信息回答问题，并给出信息源编号。
        如果无法从参考信息中找到答案，请说明无法回答。
        参考信息:
        {chr(10).join(f"[{i + 1}] {doc}" for i, doc in enumerate(context))}
        问题: {question}
        答案:"""
        messages = [{
            "role": "user",
            "content": prompt
        }]
        answer = self._ask_llm(messages)
        return answer

    def _ask_llm(self, messages):
        print("提问中……")
        response = self.client.chat.completions.create(
            model=self.chat_model,
            messages=messages,
            max_tokens=1024
        )
        answer = response.choices[0].message.content
        print(f"\n生成的答案: {answer}")
        return answer
