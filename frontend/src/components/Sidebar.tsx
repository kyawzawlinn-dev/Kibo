import { useState } from "react";
import { PlusCircle, Trash2, Edit3, Activity } from "lucide-react";
import type { Chat } from "../types";

interface SidebarProps {
  chats: Chat[];
  activeChatId: number | null;
  onNewChat: () => void;
  onSelectChat: (id: number) => void;
  onDeleteChat: (id: number) => void;
  onRenameChat: (id: number, newTitle: string) => void;
  onNavigate: (page: "chat" | "bodyrecord") => void;
}

export default function Sidebar({
  chats,
  activeChatId,
  onNewChat,
  onSelectChat,
  onDeleteChat,
  onRenameChat,
  onNavigate,
}: SidebarProps) {
  const [editingChatId, setEditingChatId] = useState<number | null>(null);
  const [editValue, setEditValue] = useState("");

  const startEdit = (id: number, title: string) => {
    setEditingChatId(id);
    setEditValue(title);
  };

  const saveEdit = (id: number) => {
    if (editValue.trim() !== "") {
      onRenameChat(id, editValue.trim());
    }
    setEditingChatId(null);
  };

  return (
    <aside className="w-64 bg-white border-r h-full flex flex-col p-4 shadow-md">
      {/* Logo */}
      <div className="text-2xl font-bold text-green-600 mb-4">🌿 Kibo</div>

      {/* Body Record Navigation */}
      <button
        onClick={() => onNavigate("bodyrecord")}
        className="flex items-center mb-4 text-gray-700 hover:text-green-600"
      >
        <Activity className="mr-2 w-5 h-5" /> Body Record
      </button>

      {/* New Chat Button */}
      <button
        onClick={onNewChat}
        className="flex items-center text-gray-700 hover:text-green-600 mb-2"
      >
        <PlusCircle className="mr-2 w-5 h-5" /> New Chat
      </button>

      {/* Recent Chats */}
      <h3 className="text-sm text-gray-500 mt-4 mb-2 uppercase">Recent Chats</h3>

      <div className="space-y-2 overflow-y-auto flex-1">
        {chats.map((chat) => {
          const isSelected = chat.id === activeChatId;

          return (
            
            <div
              key={chat.id}
              onClick={() => {
                  console.log("[DEBUG] chat item clicked:", chat.id);
                  onSelectChat(chat.id);
                              }}
              className={`flex items-center justify-between px-2 py-1 rounded cursor-pointer
                ${isSelected ? "text-green-600 font-bold" : "text-gray-700 hover:text-green-600"}`}
            >
              {editingChatId === chat.id ? (
                <input
                  className="w-full p-1 border rounded text-sm"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onBlur={() => saveEdit(chat.id)}
                  onKeyDown={(e) => e.key === "Enter" && saveEdit(chat.id)}
                  onClick={(e) => e.stopPropagation()}
                  autoFocus
                />
              ) : (
                <div className="text-left text-sm flex-1 truncate">
                  {chat.name ?? (chat as any).title ?? "Untitled"}
                </div>
              )}

              <div className="flex gap-1 ml-2">
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    startEdit(chat.id, chat.name);
                  }}
                >
                  <Edit3 size={14} className="text-gray-500 hover:text-green-600" />
                </button>

                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onDeleteChat(chat.id);
                  }}
                >
                  <Trash2 size={14} className="text-gray-500 hover:text-red-600" />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </aside>
  );
}