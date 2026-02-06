# Request/Response 与 Go 端 wordDesc 及 API 约定一致
from typing import Optional
from pydantic import BaseModel, Field


class Definition(BaseModel):
    pos: str = ""
    meaning: list[str] = Field(default_factory=list)


class Phrase(BaseModel):
    example: str = ""
    example_cn: str = ""


class WordDesc(BaseModel):
    error: str = "false"
    word: str = ""
    pronunciation: str = ""
    definitions: list[Definition] = Field(default_factory=list)
    derivatives: list[str] = Field(default_factory=list)
    exam_tags: list[str] = Field(default_factory=list)
    example: str = ""
    example_cn: str = ""
    phrases: list[Phrase] = Field(default_factory=list)
    synonyms: list[str] = Field(default_factory=list)
    llm_model_name: str = "deepseek"
    class Config:
        populate_by_name = True


# --- API Request ---
class WordDefinitionsRequest(BaseModel):
    words: list[str] = Field(default_factory=list, min_length=1, max_length=20)


# --- API Response (与 Go json.go 解析结构一致) ---
class WordDefinitionsResponse(BaseModel):
    words: list[WordDesc] = Field(default_factory=list)


class ArticleRequest(BaseModel):
    words: list[str] = Field(default_factory=list, min_length=1)


class ArticleResponse(BaseModel):
    error: str = "false"
    article: str = ""
    article_cn: str = ""
