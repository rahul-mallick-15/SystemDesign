package main

import (
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
