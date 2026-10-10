"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { Search, MailPlus, Send } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { toast } from "sonner";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import {
  useConversations,
  useConversationMessages,
  useSendMessageToConversation,
  useSendMessageToUser,
} from "@/hooks/useMessages";
import { useUIStore } from "@/store/useUIStore";
import { useAuth } from "@/hooks/useAuth";
import type { MessageResponse, UserResponse } from "@/types";
import { useSearchUsers, useSuggestedUsers } from "@/hooks/useDiscovery";
import { useDebounce } from "@/hooks/useDebounce";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

function formatRelativeTime(value: string): string {
  const at = new Date(value).getTime();
  const now = Date.now();
  const diffMinutes = Math.max(1, Math.floor((now - at) / 60000));
  if (diffMinutes < 60) return `${diffMinutes}m`;
  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d`;
}

type ActiveChat =
  | { type: "private"; id: number }
  | { type: "new_private"; user: UserResponse };

export default function MessagesPage() {
  const { user, isLoggedIn } = useAuth();

  const { openSignInModal } = useUIStore();
  const [activeChat, setActiveChat] = useState<ActiveChat | null>(null);
  const [messageInput, setMessageInput] = useState("");
  const [hasHydrated, setHasHydrated] = useState(false);

  // Restore activeChat from sessionStorage on mount when user is loaded
  useEffect(() => {
    if (!user?.id || hasHydrated) return;

    // Clean up old generic key if it exists
    sessionStorage.removeItem("twitter-clone-active-chat");

    const key = `twitter-clone-active-chat-${user.id}`;
    const savedChat = sessionStorage.getItem(key);
    if (savedChat) {
      try {
        const parsed = JSON.parse(savedChat);
        // Avoid cascading render by scheduling state update
        requestAnimationFrame(() => setActiveChat(parsed));
      } catch (e) {
        console.error("Failed to parse activeChat from sessionStorage", e);
      }
    }
    // Schedule hydrated flag change
    requestAnimationFrame(() => setHasHydrated(true));
  }, [user?.id, hasHydrated]);

  // Save activeChat to sessionStorage when it changes
  useEffect(() => {
    if (hasHydrated && user?.id) {
      const key = `twitter-clone-active-chat-${user.id}`;
      if (activeChat) {
        sessionStorage.setItem(key, JSON.stringify(activeChat));
      } else {
        sessionStorage.removeItem(key);
      }
    }
  }, [activeChat, hasHydrated, user?.id]);

  // New Message Modal State
  const [isNewMessageOpen, setIsNewMessageOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const debouncedSearch = useDebounce(searchQuery, 300);

  // DM List Search State
  const [dmSearchQuery, setDmSearchQuery] = useState("");
  const [appliedDmSearchQuery, setAppliedDmSearchQuery] = useState("");

  // Scroll refs
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const topSentinelRef = useRef<HTMLDivElement>(null);
  const prevActiveChatRef = useRef<string>("");
  const initialLoadRef = useRef<Record<string, boolean>>({});
  const preventScrollFetchRef = useRef<boolean>(true);
  const messageInputRef = useRef<HTMLInputElement>(null);

  const { data: searchData, isLoading: isSearchLoading } = useSearchUsers(
    debouncedSearch,
    isLoggedIn,
    10,
  );
  const searchResults = searchData?.items;

  const { data: suggestedUsersData, isLoading: isSuggestedLoading } =
    useSuggestedUsers(10);

  const displayUsers =
    debouncedSearch.length > 0
      ? searchResults || []
      : suggestedUsersData?.items || [];
  const isUserListLoading =
    debouncedSearch.length > 0 ? isSearchLoading : isSuggestedLoading;

  const startPrivateChatMutation = useSendMessageToUser();

  const handleStartConversation = async (selectedUser: UserResponse) => {
    const existing = conversations.find((c) => c.peer.id === selectedUser.id);
    if (existing) {
      setActiveChat({ type: "private", id: existing.id });
    } else {
      setActiveChat({ type: "new_private", user: selectedUser });
    }
    setIsNewMessageOpen(false);
    setSearchQuery("");
  };

  // 1. WebSocket Hook

  // 2. Private DMs Queries (Auth Only)
  const { data: conversationPages, isLoading: conversationsLoading } =
    useConversations();
  const conversations = conversationPages?.pages.flatMap((p) => p.items) ?? [];

  const privateId = activeChat?.type === "private" ? activeChat.id : null;
  const {
    data: dmPages,
    isLoading: dmsLoading,
    hasNextPage: dmHasNextPage,
    fetchNextPage: dmFetchNextPage,
    isFetchingNextPage: dmIsFetchingNextPage,
  } = useConversationMessages(privateId);
  const sendDMMutation = useSendMessageToConversation();

  // Derived State: sort oldest-first for natural top→bottom chat display
  const rawMessages: MessageResponse[] =
    dmPages?.pages.flatMap((p) => p.items) ?? [];
  const messagesData = [...rawMessages].sort(
    (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
  );

  const isLoadingMessages = dmsLoading;
  const hasNextPage = dmHasNextPage;
  const isFetchingNextPage = dmIsFetchingNextPage;

  const fetchNextPage = useCallback(() => {
    if (activeChat?.type === "private") {
      dmFetchNextPage();
    }
  }, [activeChat, dmFetchNextPage]);

  const currentPrivateConversation = conversations.find(
    (c) => c.id === privateId,
  );

  // Validate restored chat belongs to user once conversations are loaded
  useEffect(() => {
    if (
      isLoggedIn &&
      !conversationsLoading &&
      activeChat?.type === "private" &&
      conversations.length > 0
    ) {
      const exists = conversations.some((c) => c.id === (activeChat as { type: "private"; id: number }).id);
      if (!exists) {
        console.warn(
          "Restored chat no longer exists or belongs to another user. Clearing.",
        );
        requestAnimationFrame(() => setActiveChat(null));
      }
    }
  }, [isLoggedIn, conversationsLoading, conversations, activeChat?.type, activeChat?.type === "private" ? activeChat.id : undefined]);

  // Auto-scroll to bottom when chat changes or new messages arrive
  const activeChatKey = activeChat
    ? activeChat.type === "private"
      ? `private-${activeChat.id}`
      : `new-${activeChat.user.id}`
    : "";

  useEffect(() => {
    // Always scroll to bottom when switching chats
    if (prevActiveChatRef.current !== activeChatKey) {
      prevActiveChatRef.current = activeChatKey;
      preventScrollFetchRef.current = true;
      setTimeout(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: "instant" });
        setTimeout(() => {
          preventScrollFetchRef.current = false;
        }, 100);
      }, 50);
    }
  }, [activeChatKey]);

  // Scroll to bottom when messages load for the first time or new messages arrive
  useEffect(() => {
    if (!isLoadingMessages && messagesData.length > 0) {
      const container = messagesContainerRef.current;
      if (!container) return;

      const isFirstLoadForChat = !initialLoadRef.current[activeChatKey];
      // Only auto-scroll if we're near the bottom (within 150px) or it is the first load
      const isNearBottom =
        container.scrollHeight - container.scrollTop - container.clientHeight <
        150;

      if (isFirstLoadForChat || isNearBottom) {
        if (isFirstLoadForChat) preventScrollFetchRef.current = true;

        setTimeout(() => {
          messagesEndRef.current?.scrollIntoView({
            behavior: isFirstLoadForChat ? "instant" : "smooth",
          });
          if (isFirstLoadForChat) {
            setTimeout(() => {
              preventScrollFetchRef.current = false;
            }, 100);
          }
        }, 50);

        if (isFirstLoadForChat) {
          initialLoadRef.current[activeChatKey] = true;
        }
      }
    }
  }, [messagesData.length, isLoadingMessages, activeChatKey]);

  // IntersectionObserver for loading older messages when scrolling to top
  useEffect(() => {
    const sentinel = topSentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          if (preventScrollFetchRef.current) return;
          const container = messagesContainerRef.current;
          const prevScrollHeight = container?.scrollHeight || 0;

          fetchNextPage();

          // After fetch, restore scroll position so user doesn't jump
          requestAnimationFrame(() => {
            requestAnimationFrame(() => {
              if (container) {
                const newScrollHeight = container.scrollHeight;
                container.scrollTop += newScrollHeight - prevScrollHeight;
              }
            });
          });
        }
      },
      { root: messagesContainerRef.current, threshold: 0.1 },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const handleSendMessage = async () => {
    if (!messageInput.trim() || !activeChat) return;

    if (!isLoggedIn) {
      toast.error("You must log in to send messages.");
      return;
    }

    try {
      if (activeChat.type === "private") {
        await sendDMMutation.mutateAsync({
          conversationId: activeChat.id,
          content: messageInput,
        });
      } else if (activeChat.type === "new_private") {
        const msg = await startPrivateChatMutation.mutateAsync({
          userId: activeChat.user.id,
          content: messageInput,
        });
        setActiveChat({ type: "private", id: msg.conversationId });
      }
      setMessageInput("");
      // Scroll to bottom after sending and keep focus on input
      setTimeout(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
        messageInputRef.current?.focus();
      }, 100);
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      toast.error(error.response?.data?.error || "Failed to send message");
    }
  };

  return (
    <div className="flex h-screen max-h-screen overflow-hidden">
      {/* Sidebar: Chat List — fixed width, never expands */}
      <div
        className={`w-[320px] min-w-[280px] max-w-[360px] border-r border-border flex flex-col shrink-0 ${activeChat !== null ? "hidden md:flex" : "flex"}`}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 sticky top-0 bg-background/80 backdrop-blur-md z-10">
          <h1 className="text-xl font-bold text-foreground">Messages</h1>
          {isLoggedIn && (
            <div className="flex gap-2">
              <Dialog
                open={isNewMessageOpen}
                onOpenChange={setIsNewMessageOpen}
              >
                <DialogTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="rounded-full hover:bg-card"
                  >
                    <MailPlus className="w-5 h-5" />
                  </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-md">
                  <DialogHeader>
                    <DialogTitle>New message</DialogTitle>
                    <DialogDescription className="sr-only">
                      Search for a user to start a new direct message
                      conversation.
                    </DialogDescription>
                  </DialogHeader>
                  <div className="flex items-center space-x-2 border-b border-border pb-4">
                    <Search className="w-4 h-4 text-muted-foreground" />
                    <Input
                      placeholder="Search people"
                      className="border-0 focus-visible:ring-0 px-0 h-8"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                    />
                  </div>
                  <div className="min-h-[200px] max-h-[300px] overflow-y-auto">
                    {isUserListLoading ? (
                      <div className="text-center text-muted-foreground py-4">
                        Loading...
                      </div>
                    ) : displayUsers.length === 0 ? (
                      <div className="text-center text-muted-foreground py-4">
                        No users found
                      </div>
                    ) : (
                      <div className="space-y-2">
                        {displayUsers.map((u) => (
                          <div
                            key={u.id}
                            className="flex items-center gap-3 p-2 hover:bg-muted rounded-lg cursor-pointer transition-colors"
                            onClick={() => handleStartConversation(u)}
                          >
                            <Avatar className="w-10 h-10 shrink-0">
                              <AvatarImage src={u.avatarUrl ?? undefined} />
                              <AvatarFallback>
                                {(u.displayName || u.username)[0]}
                              </AvatarFallback>
                            </Avatar>
                            <div className="flex flex-col flex-1 min-w-0">
                              <span className="font-bold text-[15px] truncate">
                                {u.displayName || u.username}
                              </span>
                              <span className="text-muted-foreground text-[13px] truncate">
                                @{u.username}
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </DialogContent>
              </Dialog>
            </div>
          )}
        </div>

        {isLoggedIn && (
          <div className="px-4 pb-2">
            <div className="relative group">
              <div className="absolute inset-y-0 left-3 flex items-center pointer-events-none">
                <Search className="w-4 h-4 text-muted-foreground group-focus-within:text-primary" />
              </div>
              <input
                type="text"
                placeholder="Search Direct Messages"
                className="w-full bg-card text-foreground rounded-full py-2 pl-10 pr-4 outline-none border border-transparent focus:border-primary focus:bg-background placeholder-muted-foreground transition-all text-[15px]"
                value={dmSearchQuery}
                onChange={(e) => setDmSearchQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    setAppliedDmSearchQuery(dmSearchQuery);
                  }
                }}
              />
            </div>
          </div>
        )}

        {/* List */}
        <div className="flex-1 overflow-y-auto">
          {/* Private DMs */}
          {isLoggedIn ? (
            <>
              {conversationsLoading && (
                <div className="flex flex-col gap-1 p-2">
                  {[1, 2, 3, 4, 5].map((i) => (
                    <div key={i} className="flex gap-3 px-2 py-3">
                      <Skeleton className="w-10 h-10 rounded-full shrink-0" />
                      <div className="flex-1 space-y-2">
                        <Skeleton className="h-4 w-24" />
                        <Skeleton className="h-3 w-40" />
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {!conversationsLoading && conversations.length === 0 && (
                <div className="px-6 py-10 text-center">
                  <p className="font-bold text-xl mb-1">
                    Welcome to your inbox!
                  </p>
                  <p className="text-muted-foreground text-[15px] mb-6">
                    Drop a line, share a Tweet, or more with private
                    conversations between you and others on Twitter.
                  </p>
                  <Button
                    variant="default"
                    className="rounded-full bg-primary text-white hover:bg-primary/90 font-bold px-6"
                    onClick={() => setIsNewMessageOpen(true)}
                  >
                    Write a message
                  </Button>
                </div>
              )}
              {conversations
                .filter((conv) => {
                  if (!appliedDmSearchQuery) return true;
                  const lowerQuery = appliedDmSearchQuery.toLowerCase();
                  const peerName = (conv.peer.displayName || "").toLowerCase();
                  const peerUsername = (conv.peer.username || "").toLowerCase();
                  return (
                    peerName.includes(lowerQuery) ||
                    peerUsername.includes(lowerQuery)
                  );
                })
                .map((conv) => (
                  <div
                    key={conv.id}
                    onClick={() =>
                      setActiveChat({ type: "private", id: conv.id })
                    }
                    className={`flex gap-3 px-4 py-3 cursor-pointer transition-colors hover:bg-muted/50 ${activeChat?.type === "private" && activeChat?.id === conv.id ? "border-r-2 border-r-primary bg-muted/30" : ""}`}
                  >
                    <Avatar className="w-10 h-10 shrink-0">
                      <AvatarImage src={conv.peer.avatarUrl ?? undefined} />
                      <AvatarFallback>
                        {(conv.peer.displayName || conv.peer.username)[0]}
                      </AvatarFallback>
                    </Avatar>
                    <div className="flex-1 min-w-0">
                      <div className="flex justify-between items-baseline gap-2">
                        <div className="truncate text-foreground font-bold text-[15px]">
                          {conv.peer.displayName || conv.peer.username}
                          <span className="font-normal text-muted-foreground ml-1">
                            @{conv.peer.username}
                          </span>
                        </div>
                        <span className="text-muted-foreground text-[13px] whitespace-nowrap shrink-0">
                          {formatRelativeTime(conv.lastMessage.createdAt)}
                        </span>
                      </div>
                      <div className="text-muted-foreground text-[15px] truncate">
                        {conv.lastMessage.content}
                      </div>
                    </div>
                  </div>
                ))}
            </>
          ) : (
            <div className="px-6 py-10 text-center">
              <p className="font-bold text-xl mb-1">Welcome to your inbox!</p>
              <p className="text-muted-foreground text-[15px] mb-6">
                Drop a line, share a Tweet, or more with private conversations
                between you and others on Twitter.
              </p>
              <Button
                variant="default"
                className="rounded-full bg-primary text-white hover:bg-primary/90 font-bold px-6"
                onClick={() => openSignInModal()}
              >
                Log in
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Main Thread View */}
      <div
        className={`flex-1 flex flex-col min-w-0 ${activeChat ? "flex" : "hidden md:flex"}`}
      >
        {activeChat ? (
          <>
            {/* Thread Header */}
            <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-background/80 backdrop-blur-md sticky top-0 z-10">
              <div className="flex items-center gap-3 min-w-0">
                <Button
                  variant="ghost"
                  size="icon"
                  className="md:hidden -ml-2 rounded-full shrink-0"
                  onClick={() => setActiveChat(null)}
                >
                  <svg
                    viewBox="0 0 24 24"
                    aria-hidden="true"
                    className="w-5 h-5 fill-current"
                  >
                    <g>
                      <path d="M7.414 13l5.043 5.04-1.414 1.42L3.586 12l7.457-7.46 1.414 1.42L7.414 11H21v2H7.414z"></path>
                    </g>
                  </svg>
                </Button>

                {activeChat.type === "new_private" ? (
                  <div className="flex flex-col min-w-0">
                    <span className="font-bold text-[17px] text-foreground truncate">
                      {activeChat.user.displayName || activeChat.user.username}
                    </span>
                    <span className="text-[13px] text-muted-foreground truncate">
                      @{activeChat.user.username}
                    </span>
                  </div>
                ) : currentPrivateConversation ? (
                  <div className="flex flex-col min-w-0">
                    <span className="font-bold text-[17px] text-foreground truncate">
                      {currentPrivateConversation.peer.displayName ||
                        currentPrivateConversation.peer.username}
                    </span>
                    <span className="text-[13px] text-muted-foreground truncate">
                      @{currentPrivateConversation.peer.username}
                    </span>
                  </div>
                ) : null}
              </div>
            </div>

            {/* Thread Messages */}
            <div
              ref={messagesContainerRef}
              className="flex-1 overflow-y-auto p-4 flex flex-col gap-3"
            >
              <div ref={topSentinelRef} className="h-1 shrink-0" />

              {isFetchingNextPage && (
                <div className="text-muted-foreground text-center text-sm py-2">
                  Loading older messages...
                </div>
              )}

              {isLoadingMessages && activeChat.type !== "new_private" && (
                <div className="flex-1 space-y-4 py-4">
                  <div className="flex justify-start">
                    <Skeleton className="h-10 w-[60%] rounded-2xl rounded-bl-none" />
                  </div>
                  <div className="flex justify-end">
                    <Skeleton className="h-10 w-[40%] rounded-2xl rounded-br-none" />
                  </div>
                  <div className="flex justify-start">
                    <Skeleton className="h-14 w-[70%] rounded-2xl rounded-bl-none" />
                  </div>
                </div>
              )}

              {(!isLoadingMessages || activeChat.type === "new_private") &&
                messagesData.length === 0 && (
                  <div className="text-muted-foreground text-center flex-1 flex items-center justify-center">
                    <div className="max-w-xs px-4">
                      <p className="text-2xl font-bold mb-2">
                        No messages here yet...
                      </p>
                      <p className="text-muted-foreground text-sm">
                        Send a message to start the conversation.
                      </p>
                    </div>
                  </div>
                )}

              {activeChat.type !== "new_private" &&
                messagesData.map((message) => {
                  const isMine = isLoggedIn && message.sender.id === user?.id;
                  return (
                    <div
                      key={message.id}
                      className={`max-w-[85%] md:max-w-[75%] flex flex-col ${isMine ? "self-end items-end" : "self-start items-start"}`}
                    >
                      <div
                        className={`px-4 py-3 rounded-2xl text-[15px] border overflow-wrap-anywhere prose prose-invert max-w-none ${
                          isMine
                            ? "bg-primary text-primary-foreground border-primary rounded-br-none"
                            : "bg-muted/50 text-foreground border-border rounded-bl-none"
                        }`}
                      >
                        <ReactMarkdown
                          remarkPlugins={[remarkGfm]}
                          components={{
                            p: ({ children }: React.HTMLAttributes<HTMLParagraphElement>) => (
                              <p className="m-0">{children}</p>
                            ),
                            a: ({ href, children }: React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
                              <a
                                href={href}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-inherit underline decoration-current/40 hover:decoration-current"
                              >
                                {children}
                              </a>
                            ),
                          }}
                        >
                          {message.content}
                        </ReactMarkdown>
                      </div>
                      <div className="text-[11px] text-muted-foreground mt-1 px-1">
                        {formatRelativeTime(message.createdAt)}
                      </div>
                    </div>
                  );
                })}

              <div ref={messagesEndRef} className="h-0 shrink-0" />
            </div>

            {/* Thread Input */}
            <div className="p-3 border-t border-border bg-background">
              <div className="bg-card rounded-2xl flex items-center px-2 py-1">
                <Input
                  ref={messageInputRef}
                  placeholder="Start a new message"
                  className="flex-1 border-none bg-transparent focus-visible:ring-0 text-foreground placeholder-muted-foreground text-[15px]"
                  value={messageInput}
                  onChange={(e) => setMessageInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && handleSendMessage()}
                  disabled={
                    sendDMMutation.isPending ||
                    startPrivateChatMutation.isPending
                  }
                />
                <Button
                  variant="ghost"
                  size="icon"
                  className={`rounded-full w-9 h-9 transition-colors ${messageInput.trim() ? "text-primary hover:bg-primary/10" : "text-muted-foreground cursor-default hover:bg-transparent"}`}
                  onClick={handleSendMessage}
                  disabled={
                    !messageInput.trim() ||
                    sendDMMutation.isPending ||
                    startPrivateChatMutation.isPending
                  }
                >
                  <Send className="w-5 h-5" />
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div className="hidden md:flex flex-1 flex-col items-center justify-center bg-background px-4">
            <div className="max-w-[340px] text-center flex flex-col items-center">
              <h2 className="text-3xl font-bold text-foreground mb-2">
                {isLoggedIn
                  ? conversations.length > 0
                    ? "Select a message"
                    : "Welcome to your inbox!"
                  : "Welcome to your inbox!"}
              </h2>
              <p className="text-muted-foreground text-[15px] mb-6">
                {isLoggedIn
                  ? conversations.length > 0
                    ? "Choose from your existing conversations, start a new one, or just keep swimming."
                    : "Drop a line, share a Tweet, or more with private conversations between you and others on Twitter."
                  : "Drop a line, share a Tweet, or more with private conversations between you and others on Twitter."}
              </p>
              {isLoggedIn ? (
                conversations.length === 0 && (
                  <Button
                    className="rounded-full bg-primary text-primary-foreground hover:bg-primary/90 font-bold px-6 py-6 h-auto text-base"
                    onClick={() => setIsNewMessageOpen(true)}
                  >
                    Write a message
                  </Button>
                )
              ) : (
                <Button
                  className="rounded-full bg-primary text-primary-foreground hover:bg-primary/90 font-bold px-6 py-6 h-auto text-base"
                  onClick={() => openSignInModal()}
                >
                  Log in
                </Button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
