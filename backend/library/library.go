// Package library manages the user-extensible offline health library:
// the markdown articles that ground Kibo's answers. Unlike the
// emergency cards (embedded, fixed), the library lives on disk so
// users can read it, grow it, and eventually share update packs.
package library

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	logger "Kibo/backend/kibo_utils"
)

// Indexer keeps the search index in sync with the article files
// (implemented by the chat2 knowledge loader).
type Indexer interface {
	IndexFile(ctx context.Context, path string) error
	RemoveSource(ctx context.Context, source string) error
}

type Article struct {
	ID      string `json:"id"` // filename without .md; also the citation name
	Title   string `json:"title"`
	Content string `json:"content"`
}

var (
	ErrExists   = errors.New("an article with this name already exists")
	ErrInvalid  = errors.New("article needs a title and content")
	ErrNotFound = errors.New("article not found")
)

// maxArticleSize keeps a single article within sane bounds (100 KB).
const maxArticleSize = 100 * 1024

type Library struct {
	dir     string
	indexer Indexer
}

func New(dir string, indexer Indexer) *Library {
	return &Library{dir: dir, indexer: indexer}
}

// List returns all articles, sorted by title.
func (l *Library) List() ([]Article, error) {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil, fmt.Errorf("reading library folder: %w", err)
	}

	articles := make([]Article, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(l.dir, e.Name()))
		if err != nil {
			logger.Warn("[library.go/List]:\treading %s: %v", e.Name(), err)
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".md")
		articles = append(articles, Article{
			ID:      id,
			Title:   titleOf(string(raw), id),
			Content: string(raw),
		})
	}

	sort.Slice(articles, func(i, j int) bool { return articles[i].Title < articles[j].Title })
	return articles, nil
}

// Add saves a new article and indexes it so it is immediately usable
// in chat. If indexing fails the article is still kept — it will be
// indexed on the next startup.
func (l *Library) Add(ctx context.Context, title, content string) (Article, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" || len(content) > maxArticleSize {
		return Article{}, ErrInvalid
	}

	id := slugify(title)
	if id == "" {
		return Article{}, ErrInvalid
	}

	path := filepath.Join(l.dir, id+".md")
	if _, err := os.Stat(path); err == nil {
		return Article{}, ErrExists
	}

	// ensure the article starts with its title as a heading
	if !strings.HasPrefix(content, "# ") {
		content = "# " + title + "\n\n" + content
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return Article{}, fmt.Errorf("saving article: %w", err)
	}

	if err := l.indexer.IndexFile(ctx, path); err != nil {
		logger.Warn("[library.go/Add]:\tindexing %s failed (will index on restart): %v", id, err)
	}

	return Article{ID: id, Title: title, Content: content}, nil
}

// Update replaces an article's content, keeping its id (and therefore
// its citation name). The old chunks are removed from the index first
// so search never returns stale content.
func (l *Library) Update(ctx context.Context, id, content string) (Article, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > maxArticleSize {
		return Article{}, ErrInvalid
	}

	path, err := l.pathFor(id)
	if err != nil {
		return Article{}, err
	}
	if _, err := os.Stat(path); err != nil {
		return Article{}, ErrNotFound
	}

	if err := l.indexer.RemoveSource(ctx, id+".md"); err != nil {
		logger.Warn("[library.go/Update]:\tremoving old chunks of %s: %v", id, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return Article{}, fmt.Errorf("saving article: %w", err)
	}

	if err := l.indexer.IndexFile(ctx, path); err != nil {
		logger.Warn("[library.go/Update]:\tindexing %s failed (will index on restart): %v", id, err)
	}

	return Article{ID: id, Title: titleOf(content, id), Content: content}, nil
}

// Delete removes an article file and its indexed chunks.
func (l *Library) Delete(ctx context.Context, id string) error {
	path, err := l.pathFor(id)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return ErrNotFound
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting article: %w", err)
	}

	if err := l.indexer.RemoveSource(ctx, id+".md"); err != nil {
		logger.Warn("[library.go/Delete]:\tremoving chunks of %s: %v", id, err)
	}
	return nil
}

// pathFor validates an id (slug charset only — no path traversal) and
// returns its file path.
var idPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

func (l *Library) pathFor(id string) (string, error) {
	if !idPattern.MatchString(id) {
		return "", ErrInvalid
	}
	return filepath.Join(l.dir, id+".md"), nil
}

// slugify turns "Back pain" into "back_pain" — a safe filename that
// doubles as the citation name. Also blocks path traversal by
// construction (only [a-z0-9_] survive).
var slugPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(title string) string {
	s := slugPattern.ReplaceAllString(strings.ToLower(strings.TrimSpace(title)), "_")
	return strings.Trim(s, "_")
}

// titleOf extracts the first "# " heading, falling back to the id.
func titleOf(content, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return fallback
}
