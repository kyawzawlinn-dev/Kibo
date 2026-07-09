// package chat

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"path/filepath"

// 	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
// 	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef"
// )

// // VectorStorer interface for your RAG service
// type VectorStorer interface {
// 	Index(ctx context.Context, id, content string, metadata map[string]string) error
// 	Search(ctx context.Context, queryText string, k int) ([]string, error)
// }

// // ChromaVectorStore implements VectorStorer using Chroma v2
// type ChromaVectorStore struct {
// 	Client     chroma.Client
// 	Collection chroma.Collection
// }

// // CheckOrDownloadModel ensures the all-MiniLM-L6-v2 model exists locally
// func CheckOrDownloadModel() error {
// 	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache/chroma/onnx_models", "all-MiniLM-L6-v2", "onnx")
// 	tokenizerPath := filepath.Join(cacheDir, "tokenizer.json")
// 	if _, err := os.Stat(tokenizerPath); os.IsNotExist(err) {
// 		log.Println("all-MiniLM-L6-v2 model not found. You need to download it before indexing.")
// 		return err
// 		// optionally, you could automatically trigger download here if you want
// 	} else {
// 		log.Println("all-MiniLM-L6-v2 model already exists. Skipping download.")
// 	}
// 	return nil
// }

// // DeleteDuplicates scans the collection and deletes documents with duplicate text
// func (c *ChromaVectorStore) DeleteDuplicates(ctx context.Context) error {
// 	results, err := c.Collection.Query(ctx, chroma.WithQueryTexts(""), chroma.WithNResults(1000))
// 	if err != nil {
// 		return fmt.Errorf("failed to query collection: %w", err)
// 	}

// 	seen := make(map[string]chroma.DocumentID)
// 	var duplicates []chroma.DocumentID

// 	for _, group := range results.GetDocumentsGroups() {
// 		for _, doc := range group {
// 			docText := fmt.Sprintf("%v", doc)                      // serialize document text
// 			docID := chroma.DocumentID(fmt.Sprintf("%v", docText)) // NOTE: replace with actual doc ID if available

// 			if _, exists := seen[docText]; exists {
// 				duplicates = append(duplicates, docID)
// 			} else {
// 				seen[docText] = docID
// 			}
// 		}
// 	}

// 	if len(duplicates) > 0 {
// 		// Use WithDocumentIDs (correct delete option)
// 		if err := c.Collection.Delete(ctx, chroma.WithIDsDelete(duplicates...)); err != nil {
// 			return fmt.Errorf("failed to delete duplicates: %w", err)
// 		}
// 		log.Printf("🗑 Deleted %d duplicate documents", len(duplicates))
// 	} else {
// 		log.Println("No duplicate documents found")
// 	}
// 	return nil
// }

// // NewChromaVectorStore initializes Chroma client, tenant, database, and collection
// func NewChromaVectorStore(baseURL, collectionName string) (*ChromaVectorStore, error) {

// 	ctx := context.Background()

// 	client, err := chroma.NewHTTPClient(chroma.WithBaseURL(baseURL))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create Chroma client: %w", err)
// 	}

// 	// Get or create tenant
// 	tenantName := "kibo-tenant"
// 	tenant, err := client.GetTenant(ctx, chroma.NewTenant(tenantName))
// 	if err != nil {
// 		tenant, err = client.CreateTenant(ctx, chroma.NewTenant(tenantName))
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create tenant: %w", err)
// 		}
// 	}

// 	// Get or create database
// 	dbObj, err := client.GetDatabase(ctx, chroma.NewDatabase("kibo-db", tenant))
// 	if err != nil {
// 		dbObj, err = client.CreateDatabase(ctx, chroma.NewDatabase("kibo-db", tenant))
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to create database: %w", err)
// 		}
// 	}

// 	// Get or create collection
// 	col, err := client.GetOrCreateCollection(
// 		ctx,
// 		collectionName,
// 		chroma.WithDatabaseCreate(dbObj),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create or get collection: %w", err)
// 	}

// 	log.Println("✅ Chroma v2 collection initialized:", col.Name())

// 	return &ChromaVectorStore{
// 		Client:     client,
// 		Collection: col,
// 	}, nil
// }

// // Index adds a document with metadata and embedding
// func (c *ChromaVectorStore) Index(ctx context.Context, id, content string, metadata map[string]string) error {

// 	if err := CheckOrDownloadModel(); err != nil {
// 		return err
// 	}

// 	// 1️⃣ Create default embedding function
// 	ef, closeef, efErr := defaultef.NewDefaultEmbeddingFunction()
// 	if efErr != nil {
// 		return fmt.Errorf("failed to create embedding function: %w", efErr)
// 	}
// 	defer closeef()

// 	// 2️⃣ Embed the document
// 	embeddingsSlice, err := ef.EmbedDocuments(ctx, []string{content})
// 	if err != nil {
// 		return fmt.Errorf("failed to embed document: %w", err)
// 	}
// 	if len(embeddingsSlice) == 0 {
// 		return fmt.Errorf("embedding returned empty slice")
// 	}

// 	// 3️⃣ Convert metadata
// 	metaAttrs := make([]*chroma.MetaAttribute, 0, len(metadata))
// 	for k, v := range metadata {
// 		metaAttrs = append(metaAttrs, chroma.NewStringAttribute(k, v))
// 	}
// 	docMeta := chroma.NewDocumentMetadata(metaAttrs...)

// 	// 4️⃣ Add document to collection
// 	err = c.Collection.Add(
// 		ctx,
// 		chroma.WithIDs(chroma.DocumentID(id)),
// 		chroma.WithTexts(content),
// 		chroma.WithMetadatas(docMeta),
// 		chroma.WithEmbeddings(embeddingsSlice[0]), // pass the first (and only) embedding
// 	)
// 	if err != nil {
// 		return fmt.Errorf("failed to add document: %w", err)
// 	}

// 	log.Printf("✅ Indexed document ID: %s", id)
// 	return nil
// }

// // Search queries the collection for top-k documents by text
// func (c *ChromaVectorStore) Search(ctx context.Context, queryText string, k int) ([]string, error) {
// 	results, err := c.Collection.Query(
// 		ctx,
// 		chroma.WithQueryTexts(queryText),
// 		chroma.WithNResults(k),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("query failed: %w", err)
// 	}

// 	var docs []string
// 	for gi, group := range results.GetDocumentsGroups() {
// 		log.Printf("📄 Document group %d (%d docs)", gi, len(group))
// 		for _, d := range group {
// 			docText := fmt.Sprintf("%v", d)
// 			docs = append(docs, docText)
// 			log.Printf(" → Doc: %s", docText)
// 		}
// 	}

// 	return docs, nil
// }

// // InspectChromaCollection prints all documents and metadata in the collection
// func InspectChromaCollection(c *ChromaVectorStore) {
// 	ctx := context.Background()
// 	log.Println("───────── CHROMA COLLECTION INSPECTOR ──────────")
// 	log.Printf("📁 Collection: %s", c.Collection.Name())

// 	results, err := c.Collection.Query(
// 		ctx,
// 		chroma.WithQueryTexts(""),
// 		chroma.WithNResults(100), // fetch all
// 	)
// 	if err != nil {
// 		log.Println("Failed to query collection:", err)
// 		return
// 	}

//		for gi, group := range results.GetDocumentsGroups() {
//			log.Printf("📄 Document group %d (%d docs)", gi, len(group))
//			for _, d := range group {
//				docText := fmt.Sprintf("%v", d)
//				log.Printf(" → %s", docText)
//			}
//		}
//		log.Println("───────── END INSPECTOR ──────────")
//	}
package chat

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

	// Convert metadata
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

// -----------------------------
// Inspector - prints what's in the collection
// -----------------------------
func InspectChromaCollection(store *ChromaVectorStore) {
	ctx := context.Background()
	log.Println("───────── CHROMA COLLECTION INSPECTOR ──────────")
	// Print collection name
	if store.Collection.Name != nil {
		log.Printf("📁 Collection: %s", store.Collection.Name)
	} else {
		// fallback
		log.Printf("📁 Collection (unknown name)")
	}

	// Query many (empty query text => returns groups)
	res, err := store.Collection.Query(ctx, chroma.WithQueryTexts(""), chroma.WithNResults(200))
	if err != nil {
		log.Println("Failed to query collection:", err)
		return
	}

	for gi, group := range res.GetDocumentsGroups() {
		log.Printf("📄 Document group %d (%d docs)", gi, len(group))
		for _, d := range group {
			text := documentToText(d)
			id := documentToID(d)
			meta := documentToMetadata(d)
			log.Printf(" → Doc: ID=%s Text=%q Meta=%v", id, truncateForLog(text, 200), meta)
		}
	}
	log.Println("───────── END INSPECTOR ──────────")
}

// -----------------------------
// Delete duplicates by document text
// -----------------------------
// This function will find duplicated documents by text and delete all but first occurrence.
func DeleteDuplicateDocuments(store *ChromaVectorStore) (deletedCount int, err error) {
	ctx := context.Background()
	res, err := store.Collection.Query(ctx, chroma.WithQueryTexts(""), chroma.WithNResults(1000))
	if err != nil {
		return 0, fmt.Errorf("failed to query collection for dedupe: %w", err)
	}

	// map text => list of docIDs
	dupMap := map[string][]chroma.DocumentID{}
	for _, group := range res.GetDocumentsGroups() {
		for _, d := range group {
			txt := documentToText(d)
			if txt == "" {
				continue
			}
			id := documentToID(d)
			if id == "" {
				continue
			}
			dupMap[txt] = append(dupMap[txt], chroma.DocumentID(id))
		}
	}

	toDelete := []chroma.DocumentID{}
	for txt, ids := range dupMap {
		if len(ids) <= 1 {
			continue
		}
		// keep first, delete the rest
		keep := ids[0]
		log.Printf("Found %d duplicates for text (keep %s): %s\n", len(ids), keep, truncateForLog(txt, 80))
		for _, id := range ids[1:] {
			toDelete = append(toDelete, id)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Delete in batches (chroma.WithIDs is variadic)
	// Build slice of interface{} args to pass to Delete as options
	// But Delete expects options of type chroma.CollectionDeleteOption: we use chroma.WithIDs(...)
	// chroma.WithIDs accepts variadic DocumentID, so pass the full list.
	if err := store.Collection.Delete(ctx, chroma.WithIDsDelete(toDelete...)); err != nil {
		return 0, fmt.Errorf("failed to delete duplicates: %w", err)
	}

	return len(toDelete), nil
}

// -----------------------------
// Helpers: robust extractors
// -----------------------------

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

// documentToID attempts to extract an ID from the returned doc value.
func documentToID(doc interface{}) string {
	// 1) method DocumentID() string
	if getter, ok := any(doc).(interface{ DocumentID() string }); ok {
		return getter.DocumentID()
	}
	// 2) method ID() string
	if getter, ok := any(doc).(interface{ ID() string }); ok {
		return getter.ID()
	}
	// 3) reflect: fields "ID", "Id", "DocumentID"
	v := reflect.ValueOf(doc)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.IsValid() && v.Kind() == reflect.Struct {
		for _, fname := range []string{"ID", "Id", "DocumentID"} {
			f := v.FieldByName(fname)
			if f.IsValid() && f.Kind() == reflect.String {
				return f.String()
			}
		}
	}
	// 4) map
	if m, ok := doc.(map[string]interface{}); ok {
		for _, key := range []string{"id", "ID", "document_id", "documentId"} {
			if val, exists := m[key]; exists {
				if s, ok := val.(string); ok {
					return s
				}
			}
		}
	}
	// fallback empty
	return ""
}

// documentToMetadata attempts best-effort extraction of metadata.
func documentToMetadata(doc interface{}) map[string]interface{} {
	// try known method
	if getter, ok := any(doc).(interface{ Metadata() map[string]interface{} }); ok {
		return getter.Metadata()
	}
	// reflect
	v := reflect.ValueOf(doc)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.IsValid() && v.Kind() == reflect.Struct {
		if f := v.FieldByName("Metadata"); f.IsValid() {
			return mapifyValue(f.Interface())
		}
	}
	// if map
	if m, ok := doc.(map[string]interface{}); ok {
		// try to pick metadata-like key
		for _, key := range []string{"metadata", "meta", "Metadata"} {
			if val, exists := m[key]; exists {
				return mapifyValue(val)
			}
		}
	}
	return nil
}

func mapifyValue(v interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	if mv, ok := v.(map[string]interface{}); ok {
		return mv
	}
	// reflect struct -> extract string fields
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			f := rt.Field(i)
			fv := rv.Field(i)
			// only marshal simple kinds
			if fv.Kind() == reflect.String {
				out[f.Name] = fv.String()
			} else if fv.Kind() == reflect.Int || fv.Kind() == reflect.Int64 {
				out[f.Name] = fv.Int()
			}
		}
	}
	return out
}

func truncateForLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
