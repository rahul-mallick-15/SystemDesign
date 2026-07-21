package main

import (
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
