# 调用大模型（DeepSeek / OpenAI 兼容 / Gemini），与 Go LLM/models.go 行为一致
import os
from typing import Optional
from dotenv import load_dotenv
from openai import OpenAI
from schemas import WordDefinitionsResponse
from schemas import ArticleResponse
load_dotenv("../.env")

# 可选：openai 兼容（DeepSeek / OpenAI）




def _strip_json_block(text: str) -> str:
    """去掉 ```json ... ``` 包裹，与 Go processJson 一致"""
    text = text.strip()
    if text.startswith("```json") and text.endswith("```"):
        text = text[7:-3].strip()
    elif text.startswith("```") and text.endswith("```"):
        text = text[3:-3].strip()
    return text


def _process_json(raw: str) -> str:
    """解析 LLM 返回的 JSON,得到 {"words": [...]}"""
    raw = _strip_json_block(raw)
    # 兼容只返回 "words":[...] 的情况，补成完整 JSON
    if raw.lstrip().startswith('"words"'):
        raw = "{" + raw + "}"
    return raw

def call_llm(prompt: str, provider: Optional[str] = None) -> str:
    """
    调用大模型，返回纯文本。
    provider: deepseek | openai | gemini,默认取环境变量 LLM_PROVIDER 或 deepseek
    """
    provider = (provider or os.getenv("LLM_PROVIDER", "deepseek")).lower()
    api_key = os.getenv("DEEPSEEK_API_KEY") or os.getenv("OPENAI_API_KEY")
    if not api_key:
        raise ValueError("set DEEPSEEK_API_KEY or OPENAI_API_KEY")

    if provider in ("deepseek", "openai") and OpenAI is not None:
        base_url = "https://api.deepseek.com" if provider == "deepseek" else None
        client = OpenAI(api_key=api_key, base_url=base_url)
        model = "deepseek-chat" if provider == "deepseek" else "gpt-4o-mini"
        resp = client.chat.completions.create(
            model=model,
            messages=[{"role": "user", "content": prompt}],
        )
        return resp.choices[0].message.content or ""

    if provider == "gemini":
        try:
            import google.generativeai as genai
            genai.configure(api_key=os.getenv("GEMINI_API_KEY"))
            model = genai.GenerativeModel("gemini-2.0-flash")
            r = model.generate_content(prompt)
            return r.text or ""
        except Exception as e:
            raise RuntimeError(f"Gemini call failed: {e}") from e

    raise ValueError(f"unsupported LLM_PROVIDER: {provider}")


def get_word_definitions(words: list[str], provider: Optional[str] = None) -> WordDefinitionsResponse:
    """请求单词释义，返回 {"words": [WordDesc, ...]} 结构，与 Go 解析一致"""
    from prompts import get_word_prompt
    prompt = get_word_prompt(words)
    raw = call_llm(prompt, provider=provider)
    raw = _process_json(raw)

    return WordDefinitionsResponse.model_validate_json(raw)


def get_article(words: list[str], provider: Optional[str] = None) -> ArticleResponse:
    """请求文章生成，返回 {"error": "...", "article": "...", "article_cn": "..."}"""
    from prompts import get_article_prompt
    prompt = get_article_prompt(words)
    raw = call_llm(prompt, provider=provider)
    raw = _process_json(raw)
    return ArticleResponse.model_validate_json(raw)
