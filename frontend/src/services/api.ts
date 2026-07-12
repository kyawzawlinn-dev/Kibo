import type { BodyRecord, ChatResponse, DietRecord, EmergencyCard, LibraryArticle, Profile, HealthLogEntry } from "../types";

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
   PROFILES (device-trust, no passwords)
   Every API call carries the active profile in a header; the backend
   falls back to the default profile when it is missing.
------------------------------ */
const PROFILE_KEY = "kibo_profile";

export function getActiveProfileId(): number | null {
  const v = localStorage.getItem(PROFILE_KEY);
  return v ? Number(v) : null;
}

export function setActiveProfileId(id: number) {
  localStorage.setItem(PROFILE_KEY, String(id));
}

function apiFetch(input: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  const pid = localStorage.getItem(PROFILE_KEY);
  if (pid) headers.set("X-Kibo-Profile", pid);
  return fetch(input, { ...init, headers });
}

export async function getProfiles(): Promise<Profile[]> {
  const res = await apiFetch(`${API_BASE_URL}/profiles`);
  if (!res.ok) throw new Error("Failed to fetch profiles");
  return res.json();
}

export async function createProfile(name: string): Promise<Profile> {
  const res = await apiFetch(`${API_BASE_URL}/profiles`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt.trim() || "Failed to create profile");
  }
  return res.json();
}

export async function deleteProfile(id: number): Promise<void> {
  const res = await apiFetch(`${API_BASE_URL}/profiles/${id}`, { method: "DELETE" });
  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt.trim() || "Failed to delete profile");
  }
}

/* -----------------------------
   CREATE NEW CHAT
------------------------------ */
export async function createNewChat(): Promise<NewChatResponse> {
  const res = await apiFetch(`${API_BASE_URL}/chat/new`, {
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
  const res = await apiFetch(`${API_BASE_URL}/chats`);
  if (!res.ok) throw new Error("Failed to fetch chats");
  return res.json();
}

/* -----------------------------
   GET CHAT HISTORY
------------------------------ */
export async function getChatHistory(chatID: number): Promise<ChatHistoryResponse> {
  const res = await apiFetch(`${API_BASE_URL}/chat/${chatID}`);

  if (!res.ok) {
    const txt = await res.text();
    console.error("[ERROR][API] /chat failed:", txt);
    throw new Error("Failed to fetch chat history");
  }

  const data = await res.json();

  return data;
}

/* -----------------------------
   DELETE CHAT
------------------------------ */
export async function deleteChat(chatID: number): Promise<void> {
  const res = await apiFetch(`${API_BASE_URL}/chat/${chatID}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete chat");
}

/* -----------------------------
   SEND MESSAGE TO CHAT

------------------------------ */
export async function sendMessage(message: string, chatID: number): Promise<ChatResponse> {
  const res = await apiFetch(`${API_BASE_URL}/chat/${chatID}/message`, {
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
   HEALTH LOG (episode history)
------------------------------ */
export async function getHealthLog(): Promise<HealthLogEntry[]> {
  const res = await apiFetch(`${API_BASE_URL}/health-log`);
  if (!res.ok) throw new Error("Failed to fetch health log");
  return (await res.json()) ?? [];
}

export async function addHealthLogEntry(
  entry: Omit<HealthLogEntry, "id">
): Promise<HealthLogEntry> {
  const res = await apiFetch(`${API_BASE_URL}/health-log`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(entry),
  });
  if (!res.ok) throw new Error((await res.text()).trim() || "Failed to save episode");
  return res.json();
}

export async function updateHealthLogEntry(entry: HealthLogEntry): Promise<HealthLogEntry> {
  const res = await apiFetch(`${API_BASE_URL}/health-log/${entry.id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(entry),
  });
  if (!res.ok) throw new Error((await res.text()).trim() || "Failed to update episode");
  return res.json();
}

export async function deleteHealthLogEntry(id: number): Promise<void> {
  const res = await apiFetch(`${API_BASE_URL}/health-log/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error("Failed to delete episode");
}

/* -----------------------------
   CSV EXPORT / IMPORT
------------------------------ */
// Plain <a> downloads cannot carry headers, so the export link puts
// the profile in a query parameter instead.
export function exportRecordsUrl(): string {
  const pid = localStorage.getItem(PROFILE_KEY);
  return `${API_BASE_URL}/records/export.csv` + (pid ? `?profile=${pid}` : "");
}

export interface ImportResult {
  imported: number;
  skipped_duplicates: number;
  skipped_invalid: number;
}

export async function importRecordsCSV(csv: string): Promise<ImportResult> {
  const res = await apiFetch(`${API_BASE_URL}/records/import`, {
    method: "POST",
    headers: { "Content-Type": "text/csv" },
    body: csv,
  });

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt.trim() || "Failed to import CSV");
  }
  return res.json();
}

/* -----------------------------
   LAN SHARING
------------------------------ */
export async function getShareInfo(): Promise<{ urls: string[] }> {
  const res = await apiFetch(`${API_BASE_URL}/share`);
  if (!res.ok) throw new Error("Failed to fetch share info");
  return res.json();
}

/* -----------------------------
   HEALTH LIBRARY
------------------------------ */
export async function getLibraryArticles(): Promise<LibraryArticle[]> {
  const res = await apiFetch(`${API_BASE_URL}/library`);
  if (!res.ok) throw new Error("Failed to fetch library");
  return res.json();
}

export async function addLibraryArticle(
  title: string,
  content: string
): Promise<LibraryArticle> {
  const res = await apiFetch(`${API_BASE_URL}/library`, {
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
  const res = await apiFetch(`${API_BASE_URL}/library/${id}`, {
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
  const res = await apiFetch(`${API_BASE_URL}/library/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) throw new Error("Failed to delete article");
}

/* -----------------------------
   EMERGENCY FIRST-AID CARDS
------------------------------ */
export async function getEmergencyCards(): Promise<EmergencyCard[]> {
  const res = await apiFetch(`${API_BASE_URL}/emergency`);
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
  const res = await apiFetch(`${API_BASE_URL}/records/body`);
  if (!res.ok) throw new Error("Failed to fetch body records");
  const data = await res.json();
  return (data ?? []).map(toBodyRecord);
}

// saveDayRecords upserts a whole day's sheet. metrics maps record type
// to a value, or null to clear that metric for the day. Returns the
// full refreshed record list.
export async function saveDayRecords(
  date: string,
  metrics: Record<string, number | null>
): Promise<BodyRecord[]> {
  const res = await apiFetch(`${API_BASE_URL}/records/day`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ date, metrics }),
  });

  if (!res.ok) {
    const txt = await res.text();
    throw new Error(txt.trim() || "Failed to save records");
  }
  const data = await res.json();
  return (data ?? []).map(toBodyRecord);
}

export async function addBodyRecord(
  record: Omit<BodyRecord, "id">
): Promise<BodyRecord> {
  const res = await apiFetch(`${API_BASE_URL}/records/body`, {
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
  const res = await apiFetch(`${API_BASE_URL}/records/diet`);
  if (!res.ok) throw new Error("Failed to fetch diet records");
  const data = await res.json();
  return (data ?? []).map(toDietRecord);
}

export async function addDietRecord(
  record: Omit<DietRecord, "id" | "timestamp">
): Promise<DietRecord> {
  const res = await apiFetch(`${API_BASE_URL}/records/diet`, {
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
