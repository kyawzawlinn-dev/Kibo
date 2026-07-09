package chat2

import (
	"context"
	"fmt"
	"log"
	"reflect"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

// -----------------------------
// ChromaVectorStore (clean)
// -----------------------------
type ChromaVectorStore struct {
	Client     chroma.Client
	Collection chroma.Collection
}

// Index: add id, text and metadata to the collection.
// IMPORTANT: This assumes you created the collection in main with an embedding function
// (so Chroma will auto-embed incoming texts). Do NOT create embeddings here.
func (c *ChromaVectorStore) Index(ctx context.Context, id, content string, metadata map[string]string) error {

	// Convert metadata map to chroma MetaAttributes
	metaAttrs := make([]*chroma.MetaAttribute, 0, len(metadata))
	for k, v := range metadata {
		metaAttrs = append(metaAttrs, chroma.NewStringAttribute(k, v))
	}
	docMeta := chroma.NewDocumentMetadata(metaAttrs...)

	// Add document
	if err := c.Collection.Add(
		ctx,
		chroma.WithIDs(chroma.DocumentID(id)),
		chroma.WithTexts(content),
		chroma.WithMetadatas(docMeta),
	); err != nil {
		return fmt.Errorf("collection.Add failed: %w", err)
	}

	log.Printf("✅ Indexed document ID: %s", id)
	return nil
}

// Search: query by text (collection will embed query if EF configured).
// Returns the document text representations (stringified).
func (c *ChromaVectorStore) Search(ctx context.Context, queryText string, k int) ([]string, error) {
	opts := []chroma.CollectionQueryOption{
		chroma.WithQueryTexts(queryText),
		chroma.WithNResults(k),
	}

	results, err := c.Collection.Query(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("collection.Query failed: %w", err)
	}

	var out []string
	for _, group := range results.GetDocumentsGroups() {
		for _, d := range group {
			// try best-effort extraction of a readable text
			out = append(out, documentToText(d))
		}
	}
	return out, nil
}

// documentToText attempts several paths to extract human-readable text from a returned doc value.
func documentToText(doc interface{}) string {
	// 1) if doc implements a ContentString() method (some v2 types)
	if getter, ok := any(doc).(interface{ ContentString() string }); ok {
		return getter.ContentString()
	}
	// 2) method String()
	if getter2, ok := any(doc).(fmt.Stringer); ok {
		return getter2.String()
	}
	// 3) reflect - if it's a struct with Text, Content or Document fields
	v := reflect.ValueOf(doc)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.IsValid() && v.Kind() == reflect.Struct {
		// common field names
		for _, fname := range []string{"Text", "Content", "Document", "ContentString"} {
			f := v.FieldByName(fname)
			if f.IsValid() && f.Kind() == reflect.String {
				return f.String()
			}
			// if nested struct -> try nested .Text
			if f.IsValid() && f.Kind() == reflect.Struct {
				nf := f.FieldByName("Text")
				if nf.IsValid() && nf.Kind() == reflect.String {
					return nf.String()
				}
			}
		}
	}

	// 4) if it's a map[string]interface{} try common keys
	if m, ok := doc.(map[string]interface{}); ok {
		for _, key := range []string{"text", "content", "document", "Text", "Content"} {
			if v, ok := m[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
	}

	// last resort - stringify
	return fmt.Sprintf("%v", doc)
}
