import { useEffect, useMemo, useRef, useState } from "react";
import { ArrowLeft, BookOpen, Edit3, Plus, Search, Trash2, Upload } from "lucide-react";
import {
  getLibraryArticles,
  addLibraryArticle,
  updateLibraryArticle,
  deleteLibraryArticle,
} from "../services/api";
import type { LibraryArticle } from "../types";

/**
 * Very small markdown renderer for library articles: headings and
 * list items get styling, everything else is plain paragraphs. No
 * external dependency, no HTML injection (content stays text).
 */
function ArticleBody({ content }: { content: string }) {
  return (
    <div className="space-y-2 text-sm leading-relaxed text-night-50">
      {content.split("\n").map((line, i) => {
        if (line.startsWith("# ")) return null; // title shown separately
        if (line.startsWith("## "))
          return (
            <h4 key={i} className="text-mint-soft font-medium text-base pt-3">
              {line.slice(3)}
            </h4>
          );
        if (line.startsWith("- "))
          return (
            <p key={i} className="pl-4 relative">
              <span className="absolute left-0 text-mint">•</span>
              {line.slice(2)}
            </p>
          );
        if (line.trim() === "") return null;
        return <p key={i}>{line}</p>;
      })}
    </div>
  );
}

export default function Library() {
  const [articles, setArticles] = useState<LibraryArticle[]>([]);
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<LibraryArticle | null>(null);
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState("");
  const [newTitle, setNewTitle] = useState("");
  const [newContent, setNewContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  useEffect(() => {
    getLibraryArticles()
      .then(setArticles)
      .catch(() => setError("Failed to load the library."));
  }, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return articles;
    return articles.filter(
      (a) =>
        a.title.toLowerCase().includes(q) || a.content.toLowerCase().includes(q)
    );
  }, [articles, query]);

  const handleFile = (file: File) => {
    file.text().then((text) => {
      setNewContent(text);
      if (!newTitle) {
        const heading = text.split("\n").find((l) => l.startsWith("# "));
        setNewTitle(heading ? heading.slice(2).trim() : file.name.replace(/\.md$/, ""));
      }
    });
  };

  const handleSave = async () => {
    setError(null);
    setSaving(true);
    try {
      const article = await addLibraryArticle(newTitle, newContent);
      setArticles((prev) =>
        [...prev, article].sort((a, b) => a.title.localeCompare(b.title))
      );
      setAdding(false);
      setNewTitle("");
      setNewContent("");
      setSelected(article);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to save article.");
    } finally {
      setSaving(false);
    }
  };

  const handleUpdate = async () => {
    if (!selected) return;
    setError(null);
    setSaving(true);
    try {
      const article = await updateLibraryArticle(selected.id, editContent);
      setArticles((prev) =>
        prev
          .map((a) => (a.id === article.id ? article : a))
          .sort((a, b) => a.title.localeCompare(b.title))
      );
      setSelected(article);
      setEditing(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update article.");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!selected) return;
    if (!window.confirm(`Delete "${selected.title}"? Kibo will no longer use it in answers.`)) return;
    setError(null);
    try {
      await deleteLibraryArticle(selected.id);
      setArticles((prev) => prev.filter((a) => a.id !== selected.id));
      setSelected(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete article.");
    }
  };

  // --- Edit view ---
  if (selected && editing) {
    return (
      <div className="p-6 max-w-3xl mx-auto">
        <button
          onClick={() => setEditing(false)}
          className="flex items-center gap-2 text-sm text-night-400 hover:text-night-50 mb-4"
        >
          <ArrowLeft className="w-4 h-4" /> Cancel editing
        </button>
        <div className="bg-night-850 border border-night-800 rounded-xl p-6 space-y-4">
          <div>
            <h2 className="text-xl font-medium text-night-50">Edit: {selected.title}</h2>
            <p className="text-xs text-night-500 mt-1">
              The citation name ({selected.id}) stays the same. Change the "# " heading to rename the displayed title.
            </p>
          </div>

          <textarea
            value={editContent}
            onChange={(e) => setEditContent(e.target.value)}
            rows={16}
            className="w-full p-3 bg-night-900 border border-night-700 text-night-50 rounded-lg focus:outline-none focus:border-mint font-mono text-sm"
          />

          {error && <p className="text-sm text-red-400">{error}</p>}

          <button
            onClick={handleUpdate}
            disabled={saving || !editContent.trim()}
            className={`px-4 py-2 rounded-lg bg-mint text-mint-ink font-medium ${
              saving || !editContent.trim() ? "opacity-50 cursor-not-allowed" : "hover:opacity-90"
            }`}
          >
            {saving ? "Saving & reindexing…" : "Save changes"}
          </button>
        </div>
      </div>
    );
  }

  // --- Reading view ---
  if (selected) {
    return (
      <div className="p-6 max-w-3xl mx-auto">
        <button
          onClick={() => setSelected(null)}
          className="flex items-center gap-2 text-sm text-night-400 hover:text-night-50 mb-4"
        >
          <ArrowLeft className="w-4 h-4" /> Back to library
        </button>
        <div className="bg-night-850 border border-night-800 rounded-xl p-6">
          <div className="flex items-start justify-between gap-4 mb-1">
            <h2 className="text-xl font-medium text-night-50">{selected.title}</h2>
            <div className="flex gap-2 shrink-0">
              <button
                onClick={() => { setEditContent(selected.content); setEditing(true); setError(null); }}
                title="Edit article"
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-night-700 text-sm text-night-200 hover:bg-night-800"
              >
                <Edit3 className="w-3.5 h-3.5" /> Edit
              </button>
              <button
                onClick={handleDelete}
                title="Delete article"
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-night-700 text-sm text-night-200 hover:bg-night-800 hover:text-red-400"
              >
                <Trash2 className="w-3.5 h-3.5" /> Delete
              </button>
            </div>
          </div>
          <p className="text-xs text-night-500 mb-4">Cited in chat as: {selected.id}</p>
          {error && <p className="text-sm text-red-400 mb-3">{error}</p>}
          <ArticleBody content={selected.content} />
        </div>
      </div>
    );
  }

  // --- Add-article view ---
  if (adding) {
    return (
      <div className="p-6 max-w-3xl mx-auto">
        <button
          onClick={() => setAdding(false)}
          className="flex items-center gap-2 text-sm text-night-400 hover:text-night-50 mb-4"
        >
          <ArrowLeft className="w-4 h-4" /> Back to library
        </button>
        <div className="bg-night-850 border border-night-800 rounded-xl p-6 space-y-4">
          <h2 className="text-xl font-medium text-night-50">Add article</h2>
          <p className="text-sm text-night-400">
            The article becomes searchable in chat right away. Kibo will cite it
            by its name.
          </p>

          <input
            value={newTitle}
            onChange={(e) => setNewTitle(e.target.value)}
            placeholder="Title (e.g. Back pain)"
            className="w-full p-3 bg-night-900 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint"
          />

          <textarea
            value={newContent}
            onChange={(e) => setNewContent(e.target.value)}
            placeholder={"Write or paste the article here…\n\nUse '## ' for section headings and '- ' for lists."}
            rows={12}
            className="w-full p-3 bg-night-900 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint font-mono text-sm"
          />

          {error && <p className="text-sm text-red-400">{error}</p>}

          <div className="flex items-center gap-3">
            <button
              onClick={handleSave}
              disabled={saving || !newTitle.trim() || !newContent.trim()}
              className={`px-4 py-2 rounded-lg bg-mint text-mint-ink font-medium ${
                saving || !newTitle.trim() || !newContent.trim()
                  ? "opacity-50 cursor-not-allowed"
                  : "hover:opacity-90"
              }`}
            >
              {saving ? "Saving & indexing…" : "Save article"}
            </button>

            <button
              onClick={() => fileInput.current?.click()}
              className="flex items-center gap-2 px-4 py-2 rounded-lg border border-night-700 text-night-200 hover:bg-night-800"
            >
              <Upload className="w-4 h-4" /> Load .md file
            </button>
            <input
              ref={fileInput}
              type="file"
              accept=".md,.txt"
              className="hidden"
              onChange={(e) => e.target.files?.[0] && handleFile(e.target.files[0])}
            />
          </div>
        </div>
      </div>
    );
  }

  // --- List view ---
  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-2xl font-medium text-night-50 flex items-center gap-2">
          <BookOpen className="w-6 h-6 text-mint" /> Health library
        </h2>
        <button
          onClick={() => { setAdding(true); setError(null); }}
          className="flex items-center gap-2 px-3 py-2 rounded-lg bg-mint text-mint-ink text-sm font-medium hover:opacity-90"
        >
          <Plus className="w-4 h-4" /> Add article
        </button>
      </div>
      <p className="text-sm text-night-400 mb-4">
        The offline articles Kibo's answers are grounded in. Read them here, or
        add your own.
      </p>

      <div className="relative mb-4">
        <Search className="w-4 h-4 text-night-400 absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search the library..."
          className="w-full pl-9 p-3 bg-night-850 border border-night-700 text-night-50 placeholder-night-400 rounded-lg focus:outline-none focus:border-mint"
        />
      </div>

      {error && <p className="text-red-400 mb-4">{error}</p>}

      <div className="space-y-2">
        {filtered.map((article) => (
          <button
            key={article.id}
            onClick={() => setSelected(article)}
            className="w-full text-left bg-night-850 border border-night-800 rounded-xl px-5 py-4 hover:bg-night-800/60 transition-colors"
          >
            <span className="text-night-50 font-medium">{article.title}</span>
            <p className="text-xs text-night-400 mt-1 line-clamp-2">
              {article.content.split("\n").filter((l) => l && !l.startsWith("#"))[0] ?? ""}
            </p>
          </button>
        ))}
        {filtered.length === 0 && (
          <p className="text-night-500 italic p-4">No articles match your search.</p>
        )}
      </div>
    </div>
  );
}
