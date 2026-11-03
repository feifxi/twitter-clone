'use client';

import { useState } from 'react';
import { fetchWithSession } from '@/api/axiosInstance';

export interface ChatMessage {
  id: number;
  role: 'user' | 'assistant';
  content: string;
}

const GREETING: ChatMessage = {
  id: 0,
  role: 'assistant',
  content:
    "Hello! I'm ChanomBot. Ask me about what people are posting, the app, or anything else.",
};

let nextId = 1;

/** One "[event: <name>\n]data: <json>" frame of the assistant's event stream. */
function parseFrame(frame: string): { event: string; data: string | null } {
  let event = 'message';
  let data: string | null = null;
  for (const line of frame.split('\n')) {
    if (line.startsWith('event: ')) event = line.slice(7);
    else if (line.startsWith('data: ')) data = line.slice(6);
  }
  return { event, data };
}

/** A chat with the assistant for this page visit: the reply streams in as it is generated. */
export function useAssistantChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([GREETING]);
  const [streaming, setStreaming] = useState(false);

  const setReply = (id: number, content: string) =>
    setMessages((prev) => prev.map((m) => (m.id === id ? { ...m, content } : m)));

  const send = async (query: string) => {
    if (streaming) return;
    const history = messages.map((m) => ({ role: m.role === 'assistant' ? 'model' : 'user', text: m.content }));
    const replyId = nextId + 1;
    setMessages((prev) => [...prev, { id: nextId, role: 'user', content: query }, { id: replyId, role: 'assistant', content: '' }]);
    nextId += 2;
    setStreaming(true);

    try {
      const response = await fetchWithSession('/assistant', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query, history }),
      });
      if (!response.ok || !response.body) throw new Error(`assistant answered ${response.status}`);

      const reader = response.body.getReader();
      const decoder = new TextDecoder();
      let reply = '';
      let buffer = '';
      for (;;) {
        const { value, done } = await reader.read();
        buffer += decoder.decode(value, { stream: !done });
        // Frames end with a blank line; keep the trailing partial one for the next chunk
        const frames = buffer.split('\n\n');
        buffer = frames.pop() ?? '';
        for (const frame of frames) {
          const { event, data } = parseFrame(frame);
          if (data === null) continue;
          if (event === 'error') throw new Error((JSON.parse(data) as { message: string }).message);
          if (event === 'message') reply += (JSON.parse(data) as { text: string }).text;
        }
        setReply(replyId, reply);
        if (done) break;
      }
    } catch (error) {
      console.error('Assistant error:', error);
      setReply(replyId, "Sorry, I couldn't answer that. Please try again.");
    } finally {
      setStreaming(false);
    }
  };

  const clear = () => setMessages([GREETING]);

  return { messages, streaming, send, clear };
}
