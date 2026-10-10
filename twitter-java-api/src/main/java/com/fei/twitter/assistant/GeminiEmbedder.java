package com.fei.twitter.assistant;

import java.util.List;
import org.springframework.ai.embedding.EmbeddingModel;
import org.springframework.ai.embedding.EmbeddingRequest;
import org.springframework.ai.google.genai.text.GoogleGenAiTextEmbeddingOptions;
import org.springframework.ai.google.genai.text.GoogleGenAiTextEmbeddingOptions.TaskType;
import org.springframework.boot.autoconfigure.condition.ConditionalOnBean;
import org.springframework.stereotype.Component;

/** Gemini embeddings via Spring AI; only present when GEMINI_API_KEY is set. */
@Component
@ConditionalOnBean(EmbeddingModel.class)
class GeminiEmbedder implements Embedder {

    private final EmbeddingModel model;

    GeminiEmbedder(EmbeddingModel model) {
        this.model = model;
    }

    @Override
    public float[] embed(String text, Task task) {
        var options = GoogleGenAiTextEmbeddingOptions.builder()
                .taskType(task == Task.QUERY ? TaskType.RETRIEVAL_QUERY : TaskType.RETRIEVAL_DOCUMENT)
                .dimensions(DIMENSIONS)
                .build();
        float[] vector = model.call(new EmbeddingRequest(List.of(text), options)).getResult().getOutput();
        if (vector.length != DIMENSIONS) {
            throw new IllegalStateException("unexpected embedding size " + vector.length);
        }
        return vector;
    }
}
