import { useEffect, useMemo, useState } from "react";
import Sidebar from "./components/Sidebar";
import ChatBox from "./components/ChatBox";
import BodyRecord from "./components/BodyRecord";

import {
  createNewChat,
  getAllChats,
  getChatHistory,
  deleteChat,
  sendMessage,
} from "./services/api";

import type { Chat, ChatResponse, Message } from "./types";

export default function App() {
  const [chats, setChats] = useState<Chat[]>([]);
  const [activeChatId, setActiveChatId] = useState<number | null>(null);
  const [currentPage, setCurrentPage] = useState<"chat" | "bodyrecord">("chat");
  const [isLoading, setIsLoading] = useState(false);

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

      console.log("[DEBUG] /chat response:", data);

      const rawMsgs =  data.message ?? [];

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

    // STEP 3 — Append AI reply
    const aiReply: Message = {
      id: Date.now() + 1,
      text: res.reply,
      sender: "ai",
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
    <div className="flex h-screen w-screen bg-night-900 text-night-50 font-sans">
      <Sidebar
        chats={chats}
        activeChatId={activeChatId}
        currentPage={currentPage}
        onNewChat={handleNewChat}
        onSelectChat={(id) => {
          setActiveChatId(id);
          setCurrentPage("chat");
        }}
        onDeleteChat={handleDeleteChat}
        onRenameChat={handleRenameChat}
        onNavigate={(page) => setCurrentPage(page)}
      />

      <main className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="px-5 py-3 border-b border-night-800 flex items-center justify-between">
          <h1 className="text-lg font-medium text-night-50">
            {currentPage === "chat"
              ? activeChat?.name ?? "Chat"
              : "Body record"}
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
        <div className="flex-1 overflow-y-auto">
          {currentPage === "chat" && activeChat ? (
            <ChatBox
              chat={activeChat}
              onUpdateMessages={handleUpdateMessages}
              onSendMessage={handleSendMessage}
              isLoading={isLoading}
            />
          ) : currentPage === "bodyrecord" ? (
            <BodyRecord />
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

// import { useState, useMemo } from "react";
// import Sidebar from "./components/Sidebar";
// import ChatBox from "./components/ChatBox";
// import BodyRecord from "./components/BodyRecord";
// import { sendMessage } from "./services/api"; // Import the new API service
// import type{ ChatResponse, Chat, Message } from "./types";

// // Helper function to find the next available ID
// const getNextId = (chats: Chat[]) => {
//   return chats.reduce((max, chat) => (chat.id > max ? chat.id : max), 0) + 1;
// };

// // Mock Initial Data
// const initialChats: Chat[] = [
//   {
//     id: 1,
//     name: "General Health Check",
//     messages: [
//       { id: 101, text: "Hi Kibo, what is the best way to stay hydrated?", sender: "user" },
//       { id: 102, text: "The best way is to drink water regularly throughout the day. Your backend knowledge base suggests starting with 8 glasses.", sender: "ai" },
//     ],
//   },
//   { id: 2, name: "Nutrition Plan", messages: [] },
// ];

// // --- App Component ---
// export default function App() {
//   const [chats, setChats] = useState<Chat[]>(initialChats);
//   const [activeChatId, setActiveChatId] = useState<number>(initialChats[0].id);
//   const [currentPage, setCurrentPage] = useState<"chat" | "bodyrecord">("chat");
//   const [isLoading, setIsLoading] = useState(false);
  
//   // Memoize the active chat object for easy access
//   const activeChat = useMemo(() => chats.find(c => c.id === activeChatId), [chats, activeChatId]);

//   // --- Chat Management Handlers ---

//   const handleNewChat = () => {
//     const newId = getNextId(chats);
//     const newChat: Chat = {
//       id: newId,
//       name: `New Chat ${newId}`,
//       messages: [],
//     };
//     setChats(prev => [...prev, newChat]);
//     setActiveChatId(newId);
//     setCurrentPage("chat");
//   };

//   const handleSelectChat = (id: number) => {
//     setActiveChatId(id);
//     setCurrentPage("chat");
//   };

//   const handleDeleteChat = (id: number) => {
//     setChats(prev => prev.filter(c => c.id !== id));
//     // If we deleted the active chat, select the first one, or reset to 1 if empty
//     if (activeChatId === id) {
//       const remainingChats = chats.filter(c => c.id !== id);
//       if (remainingChats.length > 0) {
//         setActiveChatId(remainingChats[0].id);
//       } 
//       // else {
//       //   // If all chats are gone, create a new one
//       //   handleNewChat();
//       // }
//     }
//   };
  
//   const handleRenameChat = (id: number, newTitle: string) => {
//     setChats(prevChats => prevChats.map(chat => 
//       chat.id === id ? { ...chat, name: newTitle } : chat
//     ));
//   };
  
//   const handleNavigate = (page: "chat" | "bodyrecord") => {
//       setCurrentPage(page);
//   };

//   /**
//    * Updates the messages array for the currently active chat locally.
//    * This is typically called first by the ChatBox to show the user's message immediately.
//    */
//   const handleUpdateMessages = (updatedMessages: Message[]) => {
//     setChats(prevChats => 
//       prevChats.map(chat => 
//         chat.id === activeChatId ? { ...chat, messages: updatedMessages } : chat
//       )
//     );
//   };

//   /**
//    * Handles sending the message to the Go backend and processing the response using the API service.
//    */
//   const handleSendMessage = async (userMessage: string) => {
//     if (!activeChat) return;

//     // 1. Locally add user message (ChatBox already did this)
//     let currentMessages = activeChat.messages;
    
//     // Safety check: ensure the user message is the last one before sending
//     const lastMessage = currentMessages[currentMessages.length - 1];

//     if (!lastMessage || lastMessage.text !== userMessage || lastMessage.sender !== 'user') {
//         // Fallback: This shouldn't happen if ChatBox calls are correct
//         const newUserMsg: Message = { id: Date.now(), text: userMessage, sender: "user" };
//         currentMessages = [...currentMessages, newUserMsg];
//         handleUpdateMessages(currentMessages);
//     }
    
//     setIsLoading(true);

//     try {
//       // 2. Call the API service
//       const res: ChatResponse = await sendMessage(userMessage, activeChatId);
      
//       const aiReply: Message = { 
//           id: Date.now() + 1, // Ensure ID is unique
//           text: res.reply, 
//           sender: "ai" 
//       };

//       // 3. Update state with the AI's reply
//       const updatedMessagesWithAi = [...currentMessages, aiReply];
//       handleUpdateMessages(updatedMessagesWithAi);
      
//       // Auto-name the chat if it was newly created
//       if (currentMessages.length === 1 && userMessage.length > 5 && activeChat.name.startsWith("New Chat")) {

//            handleRenameChat(activeChatId, res.title);
//       }

//     } catch (error) {
//       console.error("Error sending message:", error);
      
//       const errorMessage: Message = { 
//           id: Date.now() + 1, 
//           text: `Connection Error: Kibo could not reach the backend API. Please check the server status. Detail: ${error instanceof Error ? error.message : 'Unknown Error'}`, 
//           sender: "ai" 
//       };
      
//       // Add error message to the chat
//       handleUpdateMessages([...currentMessages, errorMessage]);
      
//     } finally {
//       setIsLoading(false);
//     }
//   };


//   return (
//     <div className="flex h-screen w-screen bg-gray-50 text-gray-800 font-sans">
      
//       <Sidebar 
//         chats={chats}
//         activeChatId={activeChatId}
//         onNewChat={handleNewChat}
//         onSelectChat={handleSelectChat}
//         onDeleteChat={handleDeleteChat}
//         onRenameChat={handleRenameChat}
//         onNavigate={handleNavigate}
//       />

//       <main className="flex-1 flex flex-col overflow-hidden">
        
//         {/* Header/Title */}
//         <header className="p-4 border-b bg-white shadow-sm flex items-center justify-between">
//             <h1 className="text-xl font-bold text-green-700">
//                 {currentPage === "chat" 
//                     ? `💬 ${activeChat?.name || "Chat"}` 
//                     : "🏋️ Body Record"}
//             </h1>
//             {isLoading && (
//                  <div className="flex items-center text-green-500">
//                     <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-green-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
//                         <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
//                         <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
//                     </svg>
//                     Kibo is thinking...
//                 </div>
//             )}
//         </header>

//         {/* Content Area */}
//         <div className="flex-1 overflow-y-auto">
//             {currentPage === "chat" && activeChat ? (
//                 // Pass a function to ChatBox that handles the API call
//                 <ChatBox 
//                     chat={activeChat} 
//                     onUpdateMessages={handleUpdateMessages}
//                     onSendMessage={handleSendMessage}
//                     isLoading={isLoading}
//                 />
//             ) : 
//             currentPage === "bodyrecord" ? (
//                 // Render the functional BodyRecord component
//                 <BodyRecord />
//             ) : 
//             (
//                 <div className="p-8 text-center text-gray-500">
//                     Please select a chat or create a new one.
//                 </div>
//             )}
//         </div>
//       </main>
//     </div>
//   );
// }