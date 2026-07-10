export type Sender = "user" | "ai";

export type Page = "chat" | "bodyrecord" | "emergency" | "library" | "share" | "summary" | "profiles";

export interface Profile {
  id: number;
  name: string;
}

export interface EmergencyCard {
  id: string;
  title: string;
  keywords: string[];
  body: string;
}

export interface LibraryArticle {
  id: string; // citation name
  title: string;
  content: string;
}

export interface Message {
  id: number;
  text: string;
  sender: Sender;
}

export interface Chat {
  id: number;
  name: string;
  messages: Message[];
}

export interface ChatResponse {
  reply: string;
  title: string;
}

// Data models for interacting with the backend API
export interface BodyRecord {
    id?: number;
    recordType: string;
    value: number;
    unit: string;
    timestamp?: string; // ISO 8601 string
}

export interface DietRecord {
    id?: number;
    foodName: string;
    calories: number;
    protein_g?: number;
    carbs_g?: number;
    fat_g?: number;
    timestamp?: string;
}