import { Fragment, useEffect, useRef, useState } from "react";
import { NotebookPen, Check, X } from "lucide-react";
import { addHealthLogEntry } from "../services/api";
import type { Chat, Message } from "../types";

interface Props {
  chat: Chat;
  onUpdateMessages: (messages: Message[]) => void;
  onSendMessage: (message: string) => void;
  isLoading: boolean;
}

const suggestionDate = (d: string) => {
  const today = new Date();
  const iso = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}-${String(today.getDate()).padStart(2, "0")}`;
  if (d === iso) return "today";
  return new Date(`${d}T12:00:00`).toLocaleDateString();
};

export default function ChatBox({ chat, onUpdateMessages, onSendMessage, isLoading }: Props) {
  // 🔥 Use parent state only; do NOT maintain local messages state
  const messages = chat.messages;
  const [input, setInput] = useState("");
  // Per-message state for the health-log suggestion chips.
  const [handled, setHandled] = useState<Record<number, "logged" | "dismissed">>({});
  const bottomRef = useRef<HTMLDivElement>(null);

  const logFromSuggestion = async (msg: Message) => {
    const s = msg.logSuggestion;
    if (!s) return;
    try {
      await addHealthLogEntry({ date: s.date, title: s.title, severity: s.severity, notes: "From chat" });
      setHandled((prev) => ({ ...prev, [msg.id]: "logged" }));
    } catch {
      // leave the chip so the user can retry
    }
  };

  // Auto-scroll when messages change
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = () => {
    const trimmed = input.trim();
    if (!trimmed || isLoading) return;

    // 🔥 Immediately add user message to parent state
    const userMsg: Message = { id: Date.now(), text: trimmed, sender: "user" };
    onUpdateMessages([...messages, userMsg]);

    setInput("");

    // 🔥 Send message to backend via App.tsx
    onSendMessage(trimmed);
  };

  return (
    <div className="flex flex-col h-full relative">
      {/* Scrollable message area */}
      <div className="flex-1 overflow-y-auto px-4 py-4">
        <div className="mx-auto w-full max-w-xl flex flex-col gap-3">
          {messages.map((msg) => (
            <Fragment key={msg.id}>
              <div
                className={`px-4 py-2 rounded-xl break-words whitespace-pre-wrap ${
                  msg.sender === "user"
                    ? "bg-mint-deep text-night-50 self-end rounded-br-sm"
                    : "bg-night-850 border border-night-700 text-night-50 self-start rounded-tl-sm"
                }`}
              >
                {msg.text}
              </div>

              {msg.sender === "ai" &&
                msg.logSuggestion &&
                handled[msg.id] !== "dismissed" &&
                (handled[msg.id] === "logged" ? (
                  <div className="self-start flex items-center gap-2 text-xs text-mint-soft">
                    <Check className="w-3.5 h-3.5" /> Added to health log
                  </div>
                ) : (
                  <div className="self-start flex flex-wrap items-center gap-2 bg-night-900 border border-night-700 rounded-lg px-3 py-2 text-sm">
                    <NotebookPen className="w-4 h-4 text-mint-soft shrink-0" />
                    <span className="text-night-200">
                      Log <span className="text-night-50 font-medium">{msg.logSuggestion.title}</span>
                      {msg.logSuggestion.severity ? ` (${msg.logSuggestion.severity})` : ""} ·{" "}
                      {suggestionDate(msg.logSuggestion.date)}?
                    </span>
                    <button
                      onClick={() => logFromSuggestion(msg)}
                      className="px-2.5 py-1 rounded-md bg-mint text-mint-ink font-medium hover:opacity-90"
                    >
                      Add
                    </button>
                    <button
                      onClick={() => setHandled((prev) => ({ ...prev, [msg.id]: "dismissed" }))}
                      title="Dismiss"
                      className="p-1 rounded-md text-night-400 hover:bg-night-800"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
            </Fragment>
          ))}
          <div ref={bottomRef} />
        </div>
      </div>

      {/* Input area */}
      <div className="w-full p-4 flex justify-center border-t border-night-800">
        <div className="flex w-full max-w-xl gap-2">
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSend()}
            className="flex-1 bg-night-850 border border-night-700 text-night-50 placeholder-night-400 rounded-lg px-4 py-2 focus:outline-none focus:border-mint"
            placeholder="Type a message..."
            disabled={isLoading}
          />
          <button
            onClick={handleSend}
            disabled={isLoading}
            className={`bg-mint text-mint-ink font-medium px-6 py-2 rounded-lg ${
              isLoading ? "opacity-50 cursor-not-allowed" : "hover:opacity-90"
            }`}
          >
            {isLoading ? "Sending..." : "Send"}
          </button>
        </div>
      </div>
    </div>
  );
}
// import { useState, useEffect, useRef } from "react";
// import type { Chat, Message } from "../types";

// interface Props {
//   chat: Chat;
//   onUpdateMessages: (messages: Message[]) => void;
//   onSendMessage: (message: string) => void; // Added for API integration
//   isLoading: boolean; // Added for loading state
// }

// export default function ChatBox({ chat, onUpdateMessages, onSendMessage, isLoading }: Props) {
//   const [messages, setMessages] = useState<Message[]>(chat.messages);
//   const [input, setInput] = useState("");
//   const bottomRef = useRef<HTMLDivElement>(null);

//   useEffect(() => {
//     setMessages(chat.messages);
//   }, [chat.messages]);

//   // Scroll to bottom on new message
//   useEffect(() => {
//     bottomRef.current?.scrollIntoView({ behavior: "smooth" });
//   }, [messages]);

//   const handleSend = () => {
//     const trimmedInput = input.trim();
//     if (!trimmedInput || isLoading) return;

//     // 1. Add user message locally for immediate display
//     const newMsg: Message = { id: Date.now(), text: trimmedInput, sender: "user" };
//     const updated = [...messages, newMsg];
//     setMessages(updated);
//     onUpdateMessages(updated); // Notify App.tsx of the new message
    
//     // 2. Clear input and send message to App.tsx for API processing
//     setInput("");
//     onSendMessage(trimmedInput);
//   };

//  return (
//     <div className="flex flex-col h-full relative">
//       {/* Messages + Input */}
//       {messages.length === 0 ? (
//         // Center input if no messages
//         <div className="flex-1 flex justify-center items-center px-4">
//           <div className="flex w-full max-w-xl gap-2">
//             <input
//               value={input}
//               onChange={(e) => setInput(e.target.value)}
//               onKeyDown={(e) => e.key === "Enter" && handleSend()}
//               className="flex-1 border rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-green-400 dark:bg-white dark:text-gray-900 dark:border-gray-300"
//               placeholder="Type a message..."
//               disabled={isLoading}
//             />
//             <button
//               onClick={handleSend}
//               className={`bg-green-500 text-white px-6 py-2 rounded-lg transition duration-150 ${
//                 isLoading ? 'opacity-50 cursor-not-allowed' : 'hover:bg-green-600'
//               }`}
//               disabled={isLoading}
//             >
//               {isLoading ? 'Sending...' : 'Send'}
//             </button>
//           </div>
//         </div>
//       ) : (
//         // Show messages with input pinned to bottom
//         <>
//           {/* Scrollable Message Area (full width) */}
//           <div className="flex-1 overflow-y-auto px-4 py-4">
            
//             {/* Inner Wrapper: Constrained width, grows to full height, justifies content to the bottom (justify-end) */}
//             <div className="mx-auto w-full max-w-xl h-full flex flex-col justify-start">
              
//               {/* Message Content List */}
//               <div className="flex flex-col gap-3">
//                 {messages.map((msg) => (
//                   <div
//                     key={msg.id}
//                     // Clean, flat message bubble styles
//                     className={`px-4 py-2 rounded-xl break-words text-sm whitespace-pre-wrap ${
//                       msg.sender === "user"
//                         ? "bg-green-500 text-white self-end text-right rounded-br-none"
//                         : "bg-gray-100 text-gray-800 self-start text-left rounded-tl-none"
//                     }`}
//                   >
//                     {msg.text}
//                   </div>
//                 ))}
                
//                 {/* Scroll target */}
//                 <div ref={bottomRef} />
//               </div>
//             </div>
//           </div>

//           {/* Input Area (pinned to bottom, constrained to max-w-xl) */}
//           <div className="w-full p-4 flex justify-center border-t bg-white dark:bg-white">
//             <div className="flex w-full max-w-xl gap-2">
//               <input
//                 value={input}
//                 onChange={(e) => setInput(e.target.value)}
//                 onKeyDown={(e) => e.key === "Enter" && handleSend()}
//                 className="flex-1 border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-green-400 transition duration-150 dark:bg-white dark:text-gray-900 dark:border-gray-300"
//                 placeholder="Type a message..."
//                 disabled={isLoading}
//               />
//               <button
//                 onClick={handleSend}
//                 className={`bg-green-500 text-white px-6 py-2 rounded-lg transition duration-150 ${
//                     isLoading ? 'opacity-50 cursor-not-allowed' : 'hover:bg-green-600'
//                 }`}
//                 disabled={isLoading}
//               >
//                 {isLoading ? 'Sending...' : 'Send'}
//               </button>
//             </div>
//           </div>
//         </>
//       )}
//     </div>
//   );
// }