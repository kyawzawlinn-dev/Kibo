import { useState } from "react";
import {
  PlusCircle,
  Trash2,
  Edit3,
  Activity,
  AlertTriangle,
  Leaf,
  Menu,
  X,
} from "lucide-react";
import type { Chat, Page } from "../types";

interface SidebarProps {
  chats: Chat[];
  activeChatId: number | null;
  currentPage: Page;
  onNewChat: () => void;
  onSelectChat: (id: number) => void;
  onDeleteChat: (id: number) => void;
  onRenameChat: (id: number, newTitle: string) => void;
  onNavigate: (page: Page) => void;
}

export default function Sidebar({
  chats,
  activeChatId,
  currentPage,
  onNewChat,
  onSelectChat,
  onDeleteChat,
  onRenameChat,
  onNavigate,
}: SidebarProps) {
  const [expanded, setExpanded] = useState(true);
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
    <aside
      className={`shrink-0 bg-night-950 border-r border-night-800 h-full flex flex-col overflow-hidden
        transition-[width] duration-200 ease-in-out ${expanded ? "w-64" : "w-14"}`}
    >
      {/* Top: brand + close (expanded) / hamburger (collapsed) */}
      <div className="h-12 flex items-center border-b border-night-800 shrink-0">
        {expanded ? (
          <div className="w-full px-3 flex items-center justify-between">
            <span className="flex items-center gap-2 text-night-50 font-medium">
              <Leaf className="w-5 h-5 text-mint" /> Kibo
            </span>
            <button
              onClick={() => setExpanded(false)}
              aria-label="Close menu"
              className="p-1.5 rounded-md text-night-400 hover:bg-night-800 hover:text-night-50"
            >
              <X size={16} />
            </button>
          </div>
        ) : (
          <button
            onClick={() => setExpanded(true)}
            aria-label="Open menu"
            className="w-full h-full flex items-center justify-center text-night-200 hover:bg-night-800 transition-colors"
          >
            <Menu size={18} />
          </button>
        )}
      </div>

      {/* New chat */}
      <div className={`shrink-0 ${expanded ? "p-3" : "py-3 flex justify-center"}`}>
        <button
          onClick={onNewChat}
          title="New chat"
          className={`flex items-center justify-center bg-mint text-mint-ink font-medium rounded-lg hover:opacity-90 transition-opacity
            ${expanded ? "w-full px-3 py-2 gap-2" : "w-9 h-9"}`}
        >
          <PlusCircle className="w-5 h-5" />
          {expanded && "New chat"}
        </button>
      </div>

      {/* Recent chats */}
      {expanded && (
        <>
          <h3 className="text-[11px] tracking-widest text-night-500 uppercase px-4 mb-1 shrink-0">
            Recent chats
          </h3>
          <div className="space-y-0.5 overflow-y-auto flex-1 px-2 pb-2">
            {chats.map((chat) => {
              const isSelected =
                chat.id === activeChatId && currentPage === "chat";

              return (
                <div
                  key={chat.id}
                  onClick={() => onSelectChat(chat.id)}
                  className={`group flex items-center justify-between px-2 py-2 rounded-lg cursor-pointer transition-colors
                    ${
                      isSelected
                        ? "bg-mint/10 text-mint-soft font-medium"
                        : "text-night-200 hover:bg-night-800"
                    }`}
                >
                  {editingChatId === chat.id ? (
                    <input
                      className="w-full p-1 rounded text-sm bg-night-900 border border-night-700 text-night-50"
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

                  <div className="flex gap-1 ml-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      className="p-1 rounded hover:bg-night-700"
                      title="Rename chat"
                      onClick={(e) => {
                        e.stopPropagation();
                        startEdit(chat.id, chat.name);
                      }}
                    >
                      <Edit3 size={14} className="text-night-400 hover:text-mint-soft" />
                    </button>

                    <button
                      className="p-1 rounded hover:bg-night-700"
                      title="Delete chat"
                      onClick={(e) => {
                        e.stopPropagation();
                        onDeleteChat(chat.id);
                      }}
                    >
                      <Trash2 size={14} className="text-night-400 hover:text-red-400" />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </>
      )}
      {!expanded && <div className="flex-1" />}

      {/* Bottom-pinned navigation */}
      <div className={`border-t border-night-800 shrink-0 space-y-1 ${expanded ? "p-2" : "py-2 flex flex-col items-center"}`}>
        <button
          onClick={() => onNavigate("emergency")}
          title="Emergency first aid"
          className={`flex items-center rounded-lg text-sm transition-colors
            ${expanded ? "w-full gap-2.5 px-3 py-2" : "justify-center w-9 h-9"}
            ${
              currentPage === "emergency"
                ? "bg-amber-400/10 text-amber-300 font-medium"
                : "text-amber-400/90 hover:bg-night-800"
            }`}
        >
          <AlertTriangle size={16} className="shrink-0" />
          {expanded && "Emergency"}
        </button>

        <button
          onClick={() => onNavigate("bodyrecord")}
          title="Body record"
          className={`flex items-center rounded-lg text-sm transition-colors
            ${expanded ? "w-full gap-2.5 px-3 py-2" : "justify-center w-9 h-9"}
            ${
              currentPage === "bodyrecord"
                ? "bg-mint/10 text-mint-soft font-medium"
                : "text-night-200 hover:bg-night-800"
            }`}
        >
          <Activity size={16} className="shrink-0" />
          {expanded && "Body record"}
        </button>
      </div>
    </aside>
  );
}
