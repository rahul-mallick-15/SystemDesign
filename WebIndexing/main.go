package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// Document represents the raw input data coming from a web crawler.
type Document struct {
	ID      int64
	URL     string
	Content string
}

// InvertedIndex manages the search dictionary and URL mapping
type InvertedIndex struct {
	mu    sync.RWMutex       // Controls concurrent memory access (Read-Write Lock)
	store map[string][]int64 // The Inverted Index: maps a Word -> list of Document IDs
	urls  map[int64]string   // The Master Registry: maps a Document ID -> original text url
}

// ShardSearchRequest represents the query sent over the network to this node.
type ShardSearchRequest struct {
	Query string `json:"query"`
}

// ShardSearchResponse represents the local matches found by this specific node.
type ShardSearchResponse struct {
	ShardID int     `json:"shard_id"` // Helps track which server sent the data
	DocIDs  []int64 `json:"doc_ids"`  // The matching integer IDs found locally
}

func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		store: make(map[string][]int64),
		urls:  make(map[int64]string),
	}
}

// tokenize takes a raw string, cleans punctuation, and returns lowercase unique
func tokenize(text string) []string {
	// 1. Convert everything to lowercase
	lowerText := strings.ToLower(text)

	// 2. Split the text into individual words by spaces
	words := strings.Fields(lowerText)

	// 3. Track unique words to avoid duplicate entries per document
	uniqueWords := make(map[string]bool)
	var cleanedWords []string

	for _, word := range words {
		// Clean trailing/leading punctuation symbols
		cleaned := strings.Trim(word, ".,!?;:()\"'")

		// Skip empty entries or words we have already processed for this page
		if cleaned == "" || uniqueWords[cleaned] {
			continue
		}

		uniqueWords[cleaned] = true
		cleanedWords = append(cleanedWords, cleaned)
	}

	return cleanedWords
}

// IndexDocument processes a page, tracks its URL, and appends its ID to word list
func (idx *InvertedIndex) IndexDocument(doc Document) {
	// 1. Acquire a full Write Lock to prevent simultaneous read/write crashes
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 2. Save the lightweight Document ID -> URL mapping in the master registry
	idx.urls[doc.ID] = doc.URL

	// 3. Clean and isolate the words using the Tokenizer
	cleanWords := tokenize(doc.Content)

	// 4. Update the inverted index mapping for each unique word found
	for _, word := range cleanWords {
		idx.store[word] = append(idx.store[word], doc.ID)
	}
}

// Search looks up a keyword and returns a list of matching website URLs.
func (idx *InvertedIndex) Search(keyword string) []string {
	// 1. Acquire a Read Lock to allow multiple users to search simulataneous
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 2. Normalise the search keyword to lowercase
	cleanQuery := strings.ToLower(keyword)

	// 3. Look up the keyword in our inverted index dictionary
	docIDs, found := idx.store[cleanQuery]

	if !found {
		return nil // Return empty if the word doesn't exist on the internet
	}

	// 4. Translate the 64-bit integer IDs back into real text URLs
	results := make([]string, len(docIDs))
	for i, id := range docIDs {
		results[i] = idx.urls[id]
	}

	return results
}

// ServeHTTP makes InvertedIndex compatible with Go's standard web server.
func (idx *InvertedIndex) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Enforce that this endpoint only accepts HTTP POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 2. Decode the incoming JSON payload safely
	var req ShardSearchRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Query == "" {
		http.Error(w, "Bad request payload", http.StatusBadRequest)
		return
	}

	// 3. Query local memory dictionary
	idx.mu.RLock()
	cleanQuery := strings.ToLower(req.Query)
	localIDs := idx.store[cleanQuery]
	idx.mu.RUnlock()

	// 4. Package the results up into our network response format
	resp := ShardSearchResponse{
		ShardID: 1, // hardcoded for this single node
		DocIDs:  localIDs,
	}

	// 5. Send the structured JSON response back over the network connection
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	// 1. Boot up a clean, isolated database shard engine instance
	shardServer := NewInvertedIndex()

	// 2. Pre-populate our local shard database with some mock crawled web data
	shardServer.IndexDocument(Document{ID: 101, URL: "wikipedia.org/pizza", Content: "pizza history and recipe"})
	shardServer.IndexDocument(Document{ID: 102, URL: "dominos.com", Content: "order cheesy hot pizza online"})
	shardServer.IndexDocument(Document{ID: 103, URL: "italy-travel.com", Content: "visit rome for authentic history"})

	// 3. The network port this database shard will claim
	serverAddress := "127.0.0.1:8081"
	println("🚀 Shard Node #1 is online and listening at http://" + serverAddress)

	// 4. Start the blocking network listener loop
	// This listens for incoming HTTP POST search requests infinitely
	err := http.ListenAndServe(serverAddress, shardServer)
	if err != nil {
		panic("Failed to start database shard network server: " + err.Error())
	}

}
