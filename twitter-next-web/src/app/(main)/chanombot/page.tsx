"use client";

import { useState, useRef, useEffect } from "react";
import { toast } from "sonner";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Trash2, Image as ImageIcon, Send, Coffee } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useAuth } from "@/hooks/useAuth";
import { fetchWithSession } from "@/api/axiosInstance";
import { useUIStore } from "@/store/useUIStore";

type Message = {
  id: string;
  role: "user" | "assistant";
  content: string;
  isStreaming?: boolean;
};

const INITIAL_MESSAGES: Message[] = [
  {
    id: "1",
    role: "assistant",
    content:
      "Hello! I am ChanomBot, your helpful AI assistant. How can I assist you today? Whether you have questions about the app, need information, or just want to chat, I'm here to help!",
  },
];

function parseSSEFrame(frame: string): { event: string; data: string | null } {
  let event = "message";
  let data: string | null = null;
  for (const line of frame.split("\n")) {
    if (line.startsWith("event: ")) event = line.slice(7);
    else if (line.startsWith("data: ")) data = line.slice(6);
  }
  return { event, data };
}

export default function ChanomBotPage() {
  const { user, isLoggedIn } = useAuth();
  const { openSignInModal } = useUIStore();
  const [messages, setMessages] = useState<Message[]>(INITIAL_MESSAGES);
  const [input, setInput] = useState("");
  const [isClearModalOpen, setIsClearModalOpen] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const clearChat = () => {
    setMessages(INITIAL_MESSAGES);
    setIsClearModalOpen(false);
  };

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim()) return;

    if (!isLoggedIn) {
      openSignInModal();
      return;
    }

    const userContent = input.trim();
    const newUserMsg: Message = {
      id: Date.now().toString(),
      role: "user",
      content: userContent,
    };

    setMessages((prev: Message[]) => [...prev, newUserMsg]);
    setInput("");

    const history = messages.map((m: Message) => ({
      role: m.role === "assistant" ? "model" : "user",
      text: m.content,
    }));

    const botMsgId = (Date.now() + 1).toString();
    const initialBotMsg: Message = {
      id: botMsgId,
      role: "assistant",
      content: "",
      isStreaming: true,
    };
    setMessages((prev: Message[]) => [...prev, initialBotMsg]);

    try {
      const response = await fetchWithSession("/assistant", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ query: userContent, history }),
      });

      if (!response.ok) {
        throw new Error("Failed to fetch chat response");
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error("No reader available");

      const decoder = new TextDecoder();
      let done = false;
      let accumulatedContent = "";
      let sseBuffer = "";

      while (!done) {
        const { value, done: doneReading } = await reader.read();
        done = doneReading;
        sseBuffer += decoder.decode(value, { stream: !done });

        // Each frame is "[event: <name>\n]data: <json>\n\n"; keep the trailing partial frame.
        const frames = sseBuffer.split("\n\n");
        sseBuffer = frames.pop() || "";

        for (const frame of frames) {
          const { event, data } = parseSSEFrame(frame);
          if (data === null) continue;
          if (event === "error") {
            throw new Error((JSON.parse(data) as { message: string }).message);
          }
          if (event === "message") {
            accumulatedContent += (JSON.parse(data) as { text: string }).text;
          }
        }

        setMessages((prev: Message[]) =>
          prev.map((msg: Message) =>
            msg.id === botMsgId ? { ...msg, content: accumulatedContent } : msg,
          ),
        );
      }

      setMessages((prev: Message[]) =>
        prev.map((msg: Message) =>
          msg.id === botMsgId ? { ...msg, isStreaming: false } : msg,
        ),
      );
    } catch (error) {
      console.error("Chat error:", error);
      setMessages((prev: Message[]) =>
        prev.map((msg: Message) =>
          msg.id === botMsgId
            ? {
                ...msg,
                content: "Error: Failed to get response from ChanomBot.",
                isStreaming: false,
              }
            : msg,
        ),
      );
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="flex flex-col h-screen max-w-full overflow-hidden">
      {/* Header */}
      <header className="flex items-center justify-between px-4 py-3 bg-background/80 backdrop-blur-md border-b border-border shrink-0">
        <div className="flex items-center gap-4">
          <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
            <Coffee className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-bold leading-tight flex items-center gap-2">
              ChanomBot
              <span className="text-xs font-normal px-2 py-0.5 bg-accent text-accent-foreground rounded-full">
                Beta
              </span>
            </h1>
            <p className="text-xs text-muted-foreground leading-tight">
              Helpful Assistant
            </p>
          </div>
        </div>
        <Dialog open={isClearModalOpen} onOpenChange={setIsClearModalOpen}>
          <DialogTrigger asChild>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground hover:text-destructive hover:bg-destructive/10 gap-2 rounded-full"
            >
              <Trash2 className="w-4 h-4" />
              <span className="hidden sm:inline">Clear Chat</span>
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-[400px] border-border bg-background">
            <DialogHeader>
              <DialogTitle className="text-xl font-bold">
                Clear conversation?
              </DialogTitle>
              <DialogDescription className="text-muted-foreground text-[15px] pt-2">
                This will remove all messages from your current session. This
                action cannot be undone.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter className="flex flex-col sm:flex-row gap-2 pt-4">
              <Button
                variant="outline"
                onClick={() => setIsClearModalOpen(false)}
                className="rounded-full font-bold order-2 sm:order-1"
              >
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={clearChat}
                className="rounded-full bg-destructive hover:bg-destructive/90 font-bold order-1 sm:order-2"
              >
                Clear
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </header>

      {/* Messages Area */}
      <div className="flex-1 overflow-y-auto px-4 py-6 space-y-6 no-scrollbar">
        {messages.map((msg) => (
          <div key={msg.id} className="flex gap-4">
            {msg.role === "assistant" ? (
              <div className="w-10 h-10 shrink-0 rounded-full bg-primary flex items-center justify-center">
                <Coffee className="w-6 h-6 text-white" />
              </div>
            ) : (
              <Avatar className="w-10 h-10 shrink-0 border border-border/50">
                <AvatarImage src={user?.avatarUrl ?? undefined} />
                <AvatarFallback>{user?.displayName?.[0] || "U"}</AvatarFallback>
              </Avatar>
            )}
            <div className="flex flex-col flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <span className="font-bold text-[15px]">
                  {msg.role === "assistant"
                    ? "ChanomBot"
                    : user?.displayName || "You"}
                </span>
                {msg.role === "user" && (
                  <span className="text-muted-foreground text-[15px]">
                    @{user?.username || "user"}
                  </span>
                )}
              </div>
              <div className="text-[15px] leading-relaxed prose prose-invert max-w-none">
                {msg.content === "" && msg.role === "assistant" ? (
                  <div className="flex gap-1 items-center h-6">
                    <div className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce [animation-delay:-0.3s]" />
                    <div className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce [animation-delay:-0.15s]" />
                    <div className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce" />
                  </div>
                ) : (
                  <ReactMarkdown
                    remarkPlugins={[remarkGfm]}
                    components={{
                      p: ({ children }: React.HTMLAttributes<HTMLParagraphElement>) => (
                        <p className="mb-2 last:mb-0">{children}</p>
                      ),
                      a: ({ href, children }: React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
                        <a
                          href={href}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-primary hover:underline"
                        >
                          {children}
                        </a>
                      ),
                      code: ({ children }: React.HTMLAttributes<HTMLElement>) => (
                        <code className="bg-muted px-1.5 py-0.5 rounded text-sm font-mono">
                          {children}
                        </code>
                      ),
                      pre: ({ children }: React.HTMLAttributes<HTMLPreElement>) => (
                        <pre className="bg-muted p-3 rounded-lg overflow-x-auto my-2 text-sm">
                          {children}
                        </pre>
                      ),
                      ul: ({ children }: React.HTMLAttributes<HTMLUListElement>) => (
                        <ul className="list-disc ml-4 mb-2">{children}</ul>
                      ),
                      ol: ({ children }: React.HTMLAttributes<HTMLOListElement>) => (
                        <ol className="list-decimal ml-4 mb-2">{children}</ol>
                      ),
                    }}
                  >
                    {msg.content + (msg.isStreaming ? "▍" : "")}
                  </ReactMarkdown>
                )}
              </div>
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="bg-background/90 backdrop-blur-md pt-2 pb-4 px-4 border-t border-border shrink-0">
        <div className="max-w-4xl mx-auto rounded-2xl bg-accent/30 border border-border/50 focus-within:border-primary/50 focus-within:ring-1 focus-within:ring-primary/50 transition-all">
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask ChanomBot anything..."
            className="w-full max-h-48 min-h-[60px] bg-transparent resize-none outline-none p-4 text-[15px] placeholder:text-muted-foreground"
            rows={1}
            style={{ height: "auto" }}
            onInput={(e) => {
              const target = e.target as HTMLTextAreaElement;
              target.style.height = "auto";
              target.style.height = `${Math.min(target.scrollHeight, 200)}px`;
            }}
          />
          <div className="flex items-center justify-between p-2 pt-0">
            <div className="flex items-center gap-1">
              <Button
                size="icon"
                variant="ghost"
                className="rounded-full text-primary hover:bg-primary/10 h-9 w-9"
                onClick={() => toast.info("Image upload is coming soon!")}
              >
                <ImageIcon className="w-5 h-5" />
              </Button>
            </div>
            <Button
              size="icon"
              onClick={handleSend}
              disabled={!input.trim()}
              className="rounded-full bg-primary text-primary-foreground hover:bg-primary/90 h-9 w-9 disabled:opacity-50"
            >
              <Send className="w-4 h-4" />
            </Button>
          </div>
        </div>
        <div className="text-center mt-3 space-y-1">
          <p className="text-xs text-muted-foreground italic">
            💡 Note: This is a temporary session. To keep things fast, the bot
            only remembers the last 30 messages. Chat history will be cleared
            upon refresh.
          </p>
        </div>
      </div>
    </div>
  );
}
