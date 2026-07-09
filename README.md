->package needed

go get github.com/rs/cors
go get github.com/amikos-tech/chroma-go
npm install recharts

->docker
docker exec -it chroma-kibo ls /chromadb/data
docker run -p 8000:8000 -v ./chroma_data:/data --name chroma-kibo chromadb/chroma

->Project idea

User Question (e.g., "headache")
           │
           ▼
   RAGService.Ask()
           │
   ┌───────┴────────┐
   │                │
Retrieve Personal   Retrieve Knowledge
Context             Context
(from DB / stub)    (from Vector Store)
   │                │
   │                │
No recent health    simpleVectorStore.Search()
records → returns  "CARDIOLOGY REPORT...
                   LAB WORK REPORT..."
           │
           ▼
   buildPrompt(question, personalContext, knowledgeContext)
           │
           ▼
  Augmented Prompt (log)
  ┌───────────────────────────────┐
  │ User's recent health data:    │
  │ No personal health records    │
  │ General health knowledge:     │
  │ CARDIOLOGY REPORT ...         │
  │ LAB WORK REPORT ...           │
  └───────────────────────────────┘
           │
           ▼
   OllamaClient.Generate() → AI Reply


-> How to run project 
npm run dev in one tap for frontend
go run main.go in another tap for backend