import { useEffect, useMemo, useState } from "react";
import Sidebar from "./components/Sidebar";
import ChatBox from "./components/ChatBox";
import Health from "./components/Health";
import Emergency from "./components/Emergency";
import Library from "./components/Library";
import Share from "./components/Share";

import {
  createNewChat,
  getAllChats,
  getChatHistory,
  deleteChat,
  sendMessage,
  getProfiles,
  getActiveProfileId,
  setActiveProfileId,
} from "./services/api";

import type { Chat, ChatResponse, Message, Page, Profile } from "./types";
import Profiles from "./components/Profiles";

export default function App() {
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChatId, setActiveChatId] = useState<number | null>(null);
  const [currentPage, setCurrentPage] = useState<Page>("chat");
  const [isLoading, setIsLoading] = useState(false);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [activeProfileId, setActiveProfile] = useState<number | null>(
    getActiveProfileId()
  );

  const loadProfiles = async () => {
    try {
      const list = await getProfiles();
      setProfiles(list);
      // keep the stored selection if it still exists, else fall back
      const stored = getActiveProfileId();
      const valid = list.find((p) => p.id === stored) ?? list[0];
      if (valid) {
        setActiveProfile(valid.id);
        setActiveProfileId(valid.id);
      }
    } catch (err) {
      console.error("Failed to load profiles:", err);
    }
  };

  useEffect(() => {
    loadProfiles();
  }, []);

  const activeChat = useMemo(
    () => chats.find((c) => c.id === activeChatId) ?? null,
    [chats, activeChatId]
  );

  // -------------------------------------------------------
  // LOAD CHAT LIST ON MOUNT
  // -------------------------------------------------------
  useEffect(() => {
    const loadChats = async () => {
      try {
        const backendChats = await getAllChats();

        console.log("[LOAD CHATS RAW]", backendChats);

        const parsed: Chat[] = backendChats.map((chat: any, index: number) => ({
          id: chat.id ?? chat.ID,
          name: chat.title ?? chat.Title ?? `Chat ${index + 1}`,
          messages: [],
        }));

        setChats(parsed);
      } catch (err) {
        console.error("Failed to load chats:", err);
      }
    };

    loadChats();
  }, []);

  // -------------------------------------------------------
  // SELECT FIRST CHAT WHEN LIST LOADED
  // -------------------------------------------------------
  useEffect(() => {
    if (activeChatId === null && chats.length > 0) {
      setActiveChatId(chats[0].id);
    }
  }, [chats]);

  // -------------------------------------------------------
  // LOAD CHAT HISTORY WHEN activeChatId CHANGES
  // -------------------------------------------------------
  useEffect(() => {
    if (activeChatId == null) return;
    loadChatHistory(activeChatId);
  }, [activeChatId]);

  const loadChatHistory = async (id: number) => {
    try {
      const data = await getChatHistory(id);

      const rawMsgs = data.messages ?? [];

      const mapped: Message[] = rawMsgs.map((m: any, idx: number) => ({
        id: m.id ?? Date.now() + idx,
        text: m.message ?? "",
        sender: m.role === "assistant" ? "ai" : "user",
      }));

      setChats((prev) =>
        prev.map((c) => (c.id === id ? { ...c, messages: mapped } : c))
      );

      setCurrentPage("chat");
    } catch (err) {
      console.error("Failed to load chat history:", err);
    }
  };

  // -------------------------------------------------------
  // CREATE NEW CHAT
  // -------------------------------------------------------
  const handleNewChat = async () => {
    try {
      const resp = await createNewChat();
      const newChat: Chat = {
        id: resp.chat_id,
        name: resp.title || "New Chat",
        messages: [],
      };

      setChats((prev) => [...prev, newChat]);
      setActiveChatId(newChat.id);
      setCurrentPage("chat");
    } catch (err) {
      console.error("Failed to create new chat:", err);
    }
  };

  // -------------------------------------------------------
  // DELETE CHAT
  // -------------------------------------------------------
  const handleDeleteChat = async (id: number) => {
    try {
      await deleteChat(id);

      setChats((prev) => {
        const next = prev.filter((c) => c.id !== id);
        if (activeChatId === id) {
          setActiveChatId(next.length > 0 ? next[0].id : null);
        }
        return next;
      });
    } catch (err) {
      console.error("Failed to delete chat:", err);
    }
  };

  // -------------------------------------------------------
  // RENAME CHAT LOCALLY
  // -------------------------------------------------------
  const handleRenameChat = (id: number, newName: string) => {
    setChats((prev) =>
      prev.map((c) => (c.id === id ? { ...c, name: newName } : c))
    );
  };

  // -------------------------------------------------------
  // UPDATE MESSAGES (LOCAL)
  // -------------------------------------------------------
  const handleUpdateMessages = (updated: Message[]) => {
    if (activeChatId == null) return;

    setChats((prev) =>
      prev.map((c) =>
        c.id === activeChatId ? { ...c, messages: updated } : c
      )
    );
  };

  // -------------------------------------------------------
  // SEND MESSAGE TO BACKEND
  // -------------------------------------------------------
  const handleSendMessage = async (text: string) => {
  if (!activeChat) return;

  // STEP 1 — Add the user message immediately
  const userMsg: Message = {
    id: Date.now(),
    text,
    sender: "user",
  };

  const current = activeChat.messages ?? [];
  const optimisticMessages = [...current, userMsg];

  handleUpdateMessages(optimisticMessages);

  setIsLoading(true);

  try {
    // STEP 2 — Send to backend
    const res: ChatResponse = await sendMessage(text, activeChat.id);

    // STEP 3 — Append AI reply (with an optional health-log suggestion)
    const aiReply: Message = {
      id: Date.now() + 1,
      text: res.reply,
      sender: "ai",
      logSuggestion: res.log_suggestion,
    };

    handleUpdateMessages([...optimisticMessages, aiReply]);

    // Rename chat if needed
    if (current.length === 0 && res.title) {
      handleRenameChat(activeChat.id, res.title);
    }
  } catch (err) {
    console.error("Send message error:", err);

    const errorReply: Message = {
      id: Date.now(),
      text: "Connection Error: Unable to reach backend.",
      sender: "ai",
    };

    handleUpdateMessages([...optimisticMessages, errorReply]);
  } finally {
    setIsLoading(false);
  }
};

  // -------------------------------------------------------
  // RENDER UI
  // -------------------------------------------------------
  return (
    <div className="flex h-screen w-screen bg-night-900 text-night-50 font-sans print:h-auto print:bg-white">
      <Sidebar
        chats={chats}
        activeChatId={activeChatId}
        currentPage={currentPage}
        activeProfileName={
          profiles.find((p) => p.id === activeProfileId)?.name ?? null
        }
        onNewChat={handleNewChat}
        onSelectChat={(id) => {
          setActiveChatId(id);
          setCurrentPage("chat");
        }}
        onDeleteChat={handleDeleteChat}
        onRenameChat={handleRenameChat}
        onNavigate={(page) => setCurrentPage(page)}
      />

      <main className="flex-1 flex flex-col overflow-hidden print:overflow-visible">
        {/* Header */}
        <header className="px-5 py-3 border-b border-night-800 flex items-center justify-between print:hidden">
          <h1 className="text-lg font-medium text-night-50">
            {currentPage === "chat"
              ? activeChat?.name ?? "Chat"
              : currentPage === "health"
              ? "Health"
              : currentPage === "library"
              ? "Health library"
              : currentPage === "share"
              ? "Share on Wi-Fi"
              : currentPage === "profiles"
              ? "Profiles"
              : "Emergency first aid"}
          </h1>

          {isLoading && (
            <div className="flex items-center text-mint text-sm">
              <svg
                className="animate-spin -ml-1 mr-3 h-5 w-5"
                xmlns="http://www.w3.org/2000/svg"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
              Kibo is thinking...
            </div>
          )}
        </header>

        {/* Content */}
        <div className="flex-1 overflow-y-auto print:overflow-visible">
          {currentPage === "chat" && activeChat ? (
            <ChatBox
              chat={activeChat}
              onUpdateMessages={handleUpdateMessages}
              onSendMessage={handleSendMessage}
              isLoading={isLoading}
            />
          ) : currentPage === "health" ? (
            <Health />
          ) : currentPage === "emergency" ? (
            <Emergency />
          ) : currentPage === "library" ? (
            <Library />
          ) : currentPage === "share" ? (
            <Share />
          ) : currentPage === "profiles" ? (
            <Profiles
              profiles={profiles}
              activeId={activeProfileId}
              onRefresh={loadProfiles}
            />
          ) : (
            <div className="p-8 text-center text-night-400">
              Select a chat or create a new one.
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
