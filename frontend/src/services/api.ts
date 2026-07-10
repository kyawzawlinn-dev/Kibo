import type { BodyRecord, ChatResponse, DietRecord, EmergencyCard, LibraryArticle } from "../types";

export interface NewChatResponse {
  chat_id: number;
  title: string;
}

export interface ChatListItem {
  id: number;
  title: string;
  updated_at: string;
}

export interface ChatHistoryItem {
  role: string;
  message: string;
  timestamp: string;
}

export interface ChatHistoryResponse {
  chat_id: number;
  title: string;
  messages: ChatHistoryItem[];
}

// Same-origin: the Go binary serves both the UI and the API. In dev,
// Vite proxies /api to the backend (see vite.config.ts).
const API_BASE_URL = "/api";

/* -----------------------------
   CREATE NEW CHAT
------------------------------ */
export async function createNewChat(): Promise<NewChatResponse> {
  const res = await fetch(`${API_BASE_URL}/chat/new`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw new Error("Failed to create new chat");
  return res.json();
}

/* -----------------------------
   GET ALL CHATS
------------------------------ */
export async function getAllChats(): Promise<ChatListItem[]> {
  const res = await fetch(`${API_BASE_URL}/chats`);
  if (!res.ok) throw new Error("Failed to fetch chats");
  return res.json();
}

/* -----------------------------
   GET CHAT HISTORY
------------------------------ */
export async function getChatHistory(chatID: number): Promise<ChatHistoryResponse> {
  const res = await fetch(`${API_BASE_URL}/chat/${chatID}`);

  if (!res.ok) {
    const txt = await res.text();
    console.error("[ERROR][API] /chat failed:", txt);
    throw new Error("Failed to fetch chat history");
  }

  const data = await res.json();
  
  console.log("[DEBUG][API] /chat response:", data);

  return data;
}

/* -----------------------------
   DELETE CHAT
------------------------------ */
export async function deleteChat(chatID: number): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/chat/${chatID}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete chat");
}

/* -----------------------------
   SEND MESSAGE TO CHAT
   FIXED: This must call /chat/:id/message
------------------------------ */
export async function sendMessage(message: string, chatID: number): Promise<ChatResponse> {
  const res = await fetch(`${API_BASE_URL}/chat/${chatID}/message`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message }),
  });

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(`Failed to get AI reply. Status: ${res.status}. Detail: ${txt}`);
  }

  return res.json();
}

/* -----------------------------
   LAN SHARING
------------------------------ */
export async function getShareInfo(): Promise<{ urls: string[] }> {
  const res = await fetch(`${API_BASE_URL}/share`);
  if (!res.ok) throw new Error("Failed to fetch share info");
  return res.json();
}

/* -----------------------------
   HEALTH LIBRARY
------------------------------ */
export async function getLibraryArticles(): Promise<LibraryArticle[]> {
  const res = await fetch(`${API_BASE_URL}/library`);
  if (!res.ok) throw new Error("Failed to fetch library");
  return res.json();
}

export async function addLibraryArticle(
  title: string,
  content: string
): Promise<LibraryArticle> {
  const res = await fetch(`${API_BASE_URL}/library`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, content }),
  });

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt.trim() || "Failed to save article");
  }
  return res.json();
}

export async function updateLibraryArticle(
  id: string,
  content: string
): Promise<LibraryArticle> {
  const res = await fetch(`${API_BASE_URL}/library/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt.trim() || "Failed to update article");
  }
  return res.json();
}

export async function deleteLibraryArticle(id: string): Promise<void> {
  const res = await fetch(`${API_BASE_URL}/library/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete article");
}

/* -----------------------------
   EMERGENCY FIRST-AID CARDS
------------------------------ */
export async function getEmergencyCards(): Promise<EmergencyCard[]> {
  const res = await fetch(`${API_BASE_URL}/emergency`);
  if (!res.ok) throw new Error("Failed to fetch emergency cards");
  return res.json();
}

/* -----------------------------
   BODY RECORDS
   The backend uses snake_case JSON (record_type), the frontend
   camelCase (recordType) — map between them here.
------------------------------ */
function toBodyRecord(r: any): BodyRecord {
  return {
    id: r.id,
    recordType: r.record_type,
    value: r.value,
    unit: r.unit,
    timestamp: r.timestamp,
  };
}

export async function getBodyRecords(): Promise<BodyRecord[]> {
  const res = await fetch(`${API_BASE_URL}/records/body`);
  if (!res.ok) throw new Error("Failed to fetch body records");
  const data = await res.json();
  return (data ?? []).map(toBodyRecord);
}

export async function addBodyRecord(
  record: Omit<BodyRecord, "id">
): Promise<BodyRecord> {
  const res = await fetch(`${API_BASE_URL}/records/body`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      record_type: record.recordType,
      value: record.value,
      unit: record.unit,
      // optional: lets the user log a record for a past date
      ...(record.timestamp ? { timestamp: record.timestamp } : {}),
    }),
  });

  if (!res.ok) throw new Error("Failed to add body record");
  return toBodyRecord(await res.json());
}

/* -----------------------------
   DIET RECORDS
------------------------------ */
function toDietRecord(r: any): DietRecord {
  return {
    id: r.id,
    foodName: r.food_name,
    calories: r.calories,
    protein_g: r.protein,
    carbs_g: r.carbs,
    fat_g: r.fat,
    timestamp: r.timestamp,
  };
}

export async function getDietRecords(): Promise<DietRecord[]> {
  const res = await fetch(`${API_BASE_URL}/records/diet`);
  if (!res.ok) throw new Error("Failed to fetch diet records");
  const data = await res.json();
  return (data ?? []).map(toDietRecord);
}

export async function addDietRecord(
  record: Omit<DietRecord, "id" | "timestamp">
): Promise<DietRecord> {
  const res = await fetch(`${API_BASE_URL}/records/diet`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      food_name: record.foodName,
      calories: record.calories,
      protein: record.protein_g,
      carbs: record.carbs_g,
      fat: record.fat_g,
    }),
  });

  if (!res.ok) throw new Error("Failed to add diet record");
  return toDietRecord(await res.json());
}