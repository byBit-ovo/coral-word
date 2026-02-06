# gRPC 服务端：实现 LLMService，与 Go 通过 gRPC 对接
# 请在 coral_word 根目录执行: python -m llm_service.server_grpc
import os
import logging
import grpc
from concurrent import futures
from schemas import WordDesc
import coral_word_pb2
import coral_word_pb2_grpc
import llm_client

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def _word_desc_to_pb_word_desc(d: WordDesc) -> coral_word_pb2.WordDesc:
    w = coral_word_pb2.WordDesc(
        err=d.error,
        word=d.word,
        pronunciation=d.pronunciation,
        derivatives=d.derivatives,
        exam_tags=d.exam_tags,
        example=d.example,
        example_cn=d.example_cn,
        synonyms=d.synonyms,
        llm_model_name=d.llm_model_name,
    )
    for defn in d.definitions:
        w.definitions.append(
            coral_word_pb2.Definition(
                pos=defn.pos,
                meaning=defn.meaning,
            )
        )
    for ph in d.phrases:
        w.phrases.append(
            coral_word_pb2.Phrase(
                example=ph.example,
                example_cn=ph.example_cn,
            )
        )
    return w


class LLMServiceServicer(coral_word_pb2_grpc.LLMServiceServicer):
    def WordDefinitions(self, request, context):
        logger.info("LLM WordDefinitions request: %s", request.words)
        if not request.words:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("words required")
            return coral_word_pb2.WordDefinitionsResponse()
        try:
            rsp = llm_client.get_word_definitions(list(request.words))
        except Exception as e:
            logger.exception("LLM WordDefinitions error")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return coral_word_pb2.WordDefinitionsResponse()
        words = [_word_desc_to_pb_word_desc(w) for w in rsp.words]
        return coral_word_pb2.WordDefinitionsResponse(words=words)

    def Article(self, request, context):
        logger.info("LLM Article request: %s", request.words)
        if not request.words:
            context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
            context.set_details("words required")
            return coral_word_pb2.ArticleResponse()
        try:
            rsp = llm_client.get_article(list(request.words))
        except Exception as e:
            logger.exception("LLM Article error")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return coral_word_pb2.ArticleResponse()
        return coral_word_pb2.ArticleResponse(
            error=rsp.error or "",
            article=rsp.article or "",
            article_cn=rsp.article_cn or "",
        )


def serve(port: int = None):
    port = port or int(os.getenv("PYTHON_LLM_SERVICE_PORT", "50052"))
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    coral_word_pb2_grpc.add_LLMServiceServicer_to_server(LLMServiceServicer(), server)
    listen_addr = f"[::]:{port}"
    server.add_insecure_port(listen_addr)
    server.start()
    logger.info("LLM gRPC server listening on %s", listen_addr)
    server.wait_for_termination()


if __name__ == "__main__":
    serve()
