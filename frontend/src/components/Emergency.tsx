import { useEffect, useState } from "react";
import { AlertTriangle, ChevronDown, ChevronUp } from "lucide-react";
import { getEmergencyCards } from "../services/api";
import type { EmergencyCard } from "../types";

export default function Emergency() {
  const [cards, setCards] = useState<EmergencyCard[]>([]);
  const [openId, setOpenId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getEmergencyCards()
      .then(setCards)
      .catch(() => setError("Failed to load emergency cards."));
  }, []);

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h2 className="text-2xl font-medium text-night-50 mb-2 flex items-center gap-2">
        <AlertTriangle className="w-6 h-6 text-amber-400" /> Emergency first aid
      </h2>
      <p className="text-sm text-night-400 mb-6">
        Step-by-step guidance, fully offline, no AI involved. In a serious
        emergency, get professional medical help immediately.
      </p>

      {error && <p className="text-red-400">{error}</p>}

      <div className="space-y-3">
        {cards.map((card) => {
          const open = openId === card.id;
          return (
            <div
              key={card.id}
              className="bg-night-850 border border-night-800 rounded-xl overflow-hidden"
            >
              <button
                onClick={() => setOpenId(open ? null : card.id)}
                className="w-full flex items-center justify-between px-5 py-4 text-left hover:bg-night-800/60 transition-colors"
              >
                <span className="flex items-center gap-3 text-night-50 font-medium">
                  <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0" />
                  {card.title}
                </span>
                {open ? (
                  <ChevronUp className="w-4 h-4 text-night-400 shrink-0" />
                ) : (
                  <ChevronDown className="w-4 h-4 text-night-400 shrink-0" />
                )}
              </button>

              {open && (
                <div className="px-5 pb-5 border-t border-night-800 pt-4 text-night-50 text-sm leading-relaxed whitespace-pre-wrap">
                  {card.body}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
