import { useState } from "react";
import { Plus, Trash2, UsersRound } from "lucide-react";
import { createProfile, deleteProfile, setActiveProfileId } from "../services/api";
import type { Profile } from "../types";

interface ProfilesProps {
  profiles: Profile[];
  activeId: number | null;
  onRefresh: () => void;
}

export default function Profiles({ profiles, activeId, onRefresh }: ProfilesProps) {
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const switchTo = (id: number) => {
    setActiveProfileId(id);
    // Full reload: every page (chats, records, summary) re-fetches as
    // the new profile — simpler and safer than invalidating each cache.
    window.location.reload();
  };

  const handleCreate = async () => {
    if (!name.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await createProfile(name.trim());
      setName("");
      onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create profile.");
    } finally {
      setBusy(false);
    }
  };

  const handleDelete = async (profile: Profile) => {
    if (
      !window.confirm(
        `Delete profile "${profile.name}" and ALL of its chats and health records? This cannot be undone.`
      )
    )
      return;
    setError(null);
    try {
      await deleteProfile(profile.id);
      if (profile.id === activeId) {
        const other = profiles.find((p) => p.id !== profile.id);
        if (other) {
          switchTo(other.id);
          return;
        }
      }
      onRefresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete profile.");
    }
  };

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <h2 className="text-2xl font-medium text-night-50 mb-2 flex items-center gap-2">
        <UsersRound className="w-6 h-6 text-mint" /> Profiles
      </h2>
      <p className="text-sm text-night-400 mb-6">
        One device, many people. Each profile has its own chats and health
        records; the library and emergency cards are shared.
      </p>

      {error && <p className="text-red-400 mb-4">{error}</p>}

      <div className="space-y-2 mb-6">
        {profiles.map((profile) => {
          const isActive = profile.id === activeId;
          return (
            <div
              key={profile.id}
              className={`flex items-center gap-4 bg-night-850 border rounded-xl px-5 py-4 ${
                isActive ? "border-mint/40" : "border-night-800"
              }`}
            >
              <div className="w-10 h-10 rounded-full bg-mint/15 text-mint-soft flex items-center justify-center font-medium shrink-0">
                {profile.name.charAt(0).toUpperCase()}
              </div>

              <div className="flex-1 min-w-0">
                <p className="text-night-50 font-medium truncate">{profile.name}</p>
                {isActive && <p className="text-xs text-mint-soft">Active profile</p>}
              </div>

              {!isActive && (
                <button
                  onClick={() => switchTo(profile.id)}
                  className="px-3 py-1.5 rounded-lg bg-mint text-mint-ink text-sm font-medium hover:opacity-90"
                >
                  Switch
                </button>
              )}

              <button
                onClick={() => handleDelete(profile)}
                disabled={profiles.length <= 1}
                title={profiles.length <= 1 ? "Cannot delete the last profile" : "Delete profile"}
                className={`p-2 rounded-lg border border-night-700 ${
                  profiles.length <= 1
                    ? "text-night-500 opacity-40 cursor-not-allowed"
                    : "text-night-200 hover:bg-night-800 hover:text-red-400"
                }`}
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
          );
        })}
      </div>

      <div className="bg-night-850 border border-night-800 rounded-xl p-5">
        <h3 className="text-base font-medium text-night-50 mb-3">Add profile</h3>
        <div className="flex gap-2">
          <input
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCreate()}
            placeholder="Name (e.g. Mom)"
            className="flex-1 p-3 bg-night-900 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint"
          />
          <button
            onClick={handleCreate}
            disabled={busy || !name.trim()}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg bg-mint text-mint-ink font-medium ${
              busy || !name.trim() ? "opacity-50 cursor-not-allowed" : "hover:opacity-90"
            }`}
          >
            <Plus className="w-4 h-4" /> Add
          </button>
        </div>
      </div>
    </div>
  );
}
