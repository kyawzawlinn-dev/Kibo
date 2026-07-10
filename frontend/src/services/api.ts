import type { BodyRecord, ChatResponse, DietRecord } from "../types";

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

const API_BASE_URL = "http://localhost:8080/api";

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

// import type { BodyRecord, ChatResponse, DietRecord } from "../types";

// const API_BASE_URL = "http://localhost:8080/api";

// /**
//  * Sends a message to the Kibo AI Chat endpoint.
//  * @param message The user message to send.
//  * @returns The AI's reply as a string.
//  */
// export async function sendMessage(message: string, chatID: number): Promise<ChatResponse> {
//   const response = await fetch(`${API_BASE_URL}/chat`, {
//     method: "POST",
//     headers: {
//       "Content-Type": "application/json",
//     },
//     body: JSON.stringify({ message , chat_id: chatID}),
//   });

//   if (!response.ok) {
//     // If the backend failed, throw an error with the status
//     const errorText = await response.text();
//     throw new Error(`Failed to get AI reply. Status: ${response.status}. Detail: ${errorText}`);
//   }

//   // const data: { reply: string } = await response.json();
//   // return data.reply;
//     const data = await response.json();
//     return data as ChatResponse;
// }

// /**
//  * Fetches all Body Records for the user.
//  */
// export async function getBodyRecords(): Promise<BodyRecord[]> {
//   // NOTE: Assuming your Go backend exposes /api/records/body for fetching
//   const response = await fetch(`${API_BASE_URL}/records/body`);
//   if (!response.ok) throw new Error("Failed to fetch body records.");
  
//   // Mock data structure for frontend development
//   // In a real app, this would return the response.json()
//   const mockData: BodyRecord[] = [
//     { id: 1, recordType: "Weight", value: 75, unit: "kg", timestamp: "2024-01-01T10:00:00Z" },
//     { id: 2, recordType: "Sleep", value: 7.5, unit: "hours", timestamp: "2024-01-02T08:00:00Z" },
//   ];

//   return mockData;
// }

// /**
//  * Adds a new Body Record.
//  * @param record The new record data.
//  */
// export async function addBodyRecord(record: Omit<BodyRecord, 'id' | 'timestamp'>): Promise<BodyRecord> {
//   // NOTE: Assuming your Go backend exposes /api/records/body for adding
//   const response = await fetch(`${API_BASE_URL}/records/body`, {
//     method: "POST",
//     headers: { "Content-Type": "application/json" },
//     body: JSON.stringify(record),
//   });

//   if (!response.ok) throw new Error("Failed to add body record.");
  
//   // Mock return structure for the newly created record
//   return { ...record, id: Date.now(), timestamp: new Date().toISOString() };
// }

// // Placeholder for other record types (e.g., Diet)
// export async function getDietRecords(): Promise<DietRecord[]> {
//     // Implement API call to /api/records/diet
//     return [];
// }