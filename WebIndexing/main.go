package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

// PersistentIndex upgrades engine to manage disk-backed LSM structres
type PersistentIndex struct {
	mu       sync.RWMutex
	memTable map[string][]int64 // Fast, in-memory updates (Active MemTable)
	urls     map[int64]string   // Master ID -> URL registry
	dataDir  string             // The directory path on your hard drive to save index files
	walFile  *os.File           // The Write-Ahead Log file pointer
}

// NewPersistentIndex spins up a disk-backed storage engine with an active WAL log.
func NewPersistentIndex(dataDir string) (*PersistentIndex, error) {
	// 1. Create the database directory on disk if it doesn't exist
	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		return nil, err
	}

	// 2. Open or create a sequential Write-Ahead Log file (append-only mode)
	walPath := filepath.Join(dataDir, "wal.log")
	file, err := os.OpenFile(walPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &PersistentIndex{
		memTable: make(map[string][]int64),
		urls:     make(map[int64]string),
		dataDir:  dataDir,
		walFile:  file,
	}, nil
}

// SaveDocument commits a crawled document to the WAL disk log first, then updates memory.
func (p *PersistentIndex) SaveDocument(doc Document) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Format the data into a simple sequential log line
	// Format: ID,URL,Content\n
	logLine := fmt.Sprintf("%d,%s,%s\n", doc.ID, doc.URL, strings.ReplaceAll(doc.Content, "\n", " "))

	// 2. Write-Ahead Log (WAL) Phase: Commit straight to disk storage
	_, err := p.walFile.WriteString(logLine)
	if err != nil {
		return fmt.Errorf("WAL write failure, aborting index: %w", err)
	}

	// Force the operating system to immediately flush bytes to physical hardware
	p.walFile.Sync()

	// 3. MemTable Phase: Update our fast, volatile short-term RAM maps
	p.urls[doc.ID] = doc.URL
	cleanWords := tokenize(doc.Content)
	for _, word := range cleanWords {
		p.memTable[word] = append(p.memTable[word], doc.ID)
	}

	return nil
}

// FlushMemTableToDisk writes the current active RAM index to a permanent immutable file
func (p *PersistentIndex) FlushMemTableToDisk() error {
	// 1. Acquire a full Write Lock to snapshot and swap memory pointers safely
	p.mu.Lock()

	// If memory is empty, nothing to flush
	if len(p.memTable) == 0 {
		p.mu.Unlock()
		return nil
	}

	// Capture snapshots of our current memory tables
	oldMemTable := p.memTable
	oldURLs := p.urls

	// Reset memory tables to clean, empty maps for incoming writes instantly
	p.memTable = make(map[string][]int64)
	p.urls = make(map[int64]string)

	// Cycle the WAL safely INSIDE the lock boundary
	p.walFile.Close() // Close the old log

	walPath := filepath.Join(p.dataDir, "wal.log")
	os.Remove(walPath) // Safely delete the old log while incoming writes are blocked

	// Instantly create the brand new log file
	newWAL, err := os.OpenFile(walPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		p.mu.Unlock()
		return err
	}
	p.walFile = newWAL // Point the engine to the fresh log file

	// 2. Now it is safe to unlock!
	// Future writes will seamlessly type into the clean memTable and the clean newWAL.
	p.mu.Unlock()

	// 3. Prepare the unique immutable file name based on current time
	fileName := fmt.Sprintf("index_%d.db", time.Now().UnixNano())
	filePath := filepath.Join(p.dataDir, fileName)

	// Create and open the permanent disk file
	diskFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer diskFile.Close()

	// 4. Serialize and save the old snapshot data onto the disk
	// For simplicity, we package the index map and URL dictionary together as a JSON file
	type DiskSnapshot struct {
		Store map[string][]int64 `json:"store"`
		URLs  map[int64]string   `json:"urls"`
	}
	snapshot := DiskSnapshot{Store: oldMemTable, URLs: oldURLs}

	err = json.NewEncoder(diskFile).Encode(snapshot)
	if err != nil {
		return err
	}

	println("💾 MemTable successfully flushed to immutable disk file: " + fileName)
	return nil
}

// Search looks in both active memory and all unchangeable disk files
func (p *PersistentIndex) Search(keyword string) []string {
	p.mu.RLock()
	cleanQuery := strings.ToLower(keyword)

	// 1. Gather IDs from the active memory table
	var combinedIDs []int64
	combinedIDs = append(combinedIDs, p.memTable[cleanQuery]...)

	// Create a copy of the URL registry for translation later
	urlRegistryCopy := make(map[int64]string)
	for k, v := range p.urls {
		urlRegistryCopy[k] = v
	}
	p.mu.RUnlock() // Release the lock early so writes aren't blocked while we read files!

	// 2. Scan the directory for all written immutable files (*.db)
	files, _ := filepath.Glob(filepath.Join(p.dataDir, "index_*.db"))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Decode the file data
		var snapshot struct {
			Store map[string][]int64 `json:"store"`
			URLs  map[int64]string   `json:"urls"`
		}
		json.Unmarshal(data, &snapshot)

		// Append matching IDs from this disk file
		if ids, found := snapshot.Store[cleanQuery]; found {
			combinedIDs = append(combinedIDs, ids...)
		}

		// Merge the file's URL directory into our tracking map
		for id, url := range snapshot.URLs {
			urlRegistryCopy[id] = url
		}
	}

	// 3. Translate the collected unique IDs back to URLs
	var finalURLs []string
	seen := make(map[int64]bool)
	for _, id := range combinedIDs {
		if !seen[id] {
			seen[id] = true
			if url, exists := urlRegistryCopy[id]; exists {
				finalURLs = append(finalURLs, url)
			}
		}
	}
	return finalURLs
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
