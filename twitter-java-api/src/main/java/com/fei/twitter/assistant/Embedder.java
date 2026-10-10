package com.fei.twitter.assistant;

/** Turns text into a 768-dimension vector (the tweet_embeddings column size). */
public interface Embedder {

    int DIMENSIONS = 768;

    enum Task {
        /** Tweets being stored for retrieval. */
        DOCUMENT,
        /** A question being matched against stored tweets. */
        QUERY
    }

    float[] embed(String text, Task task);
}
