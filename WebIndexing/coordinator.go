package main

import (
	"context" // Add this to your import block if not present
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Coordinator handles distributing search queries to all backend shards
type Coordinator struct {
	Client    *http.Client // Reusable HTTP client connection pool
	ShardURLs []string     // List of network addresses for all database shards
}

// GlobalSearchResult represents the final combined output sent back to human user
type GlobalSearchResult struct {
	Query   string   `json:"query"`
	Results []string `json:"results"` // The final translated, text URLs
}

func (c *Coordinator) ScatterQuery(query string) []int64 {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var aggregatedIDs []int64

	// Prepare the JSON payload to send to the shards
	reqPayload := ShardSearchRequest{Query: query}
	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil
	}

	// Loop through all registered shard addresses and launch a concurrent request for each
	for _, url := range c.ShardURLs {
		wg.Add(1)

		// Launch a background Goroutine (The Scatter Phase)
		go func(targetURL string) {
			defer wg.Done()

			// 1. Issue an HTTP POST request over the network connection pool
			resp, err := c.Client.Post(targetURL, "application/json", strings.NewReader(string(jsonData)))
			if err != nil {
				return // Gracefully skip this shard if it is offline or down
			}
			defer resp.Body.Close()

			// 2. Decode the incoming shard response data
			var shardResp ShardSearchResponse
			if err := json.NewDecoder(resp.Body).Decode(&shardResp); err != nil {
				return
			}

			// 3. Thread-safely append the found local IDs into our central slice (The Gather Phase)
			mu.Lock()
			aggregatedIDs = append(aggregatedIDs, shardResp.DocIDs...)
			mu.Unlock()

		}(url) // Pass the url into the goroutine closure safely
	}

	// Block and wait until every single background network request finishes
	wg.Wait()

	return aggregatedIDs
}

// ScatterQueryWithTimeout routes queries concurrently and drops servers crossing the time limit
func (c *Coordinator) ScatterQueryWithTimeout(query string, timeout time.Duration) []int64 {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var aggregatedIDs []int64

	// 1. Create a root context that automatically cancels itself when the timeout is breached
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Always clean up resources

	reqPayload := ShardSearchRequest{Query: query}
	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return nil
	}
	for _, url := range c.ShardURLs {
		wg.Add(1)

		go func(targetURL string) {
			defer wg.Done()

			// 2. Build a brand new HTTP Request explicitly tied to timeout context
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(string(jsonData)))
			if err != nil {
				return
			}
			req.Header.Set("Context-Type", "application/json")

			// 3. Execute the request. If the context deadline passes, this call aborts immediately
			resp, err := c.Client.Do(req)
			if err != nil {
				// This catches network failures AND context deadline exceeded errors gracefully
				return
			}
			defer resp.Body.Close()

			var shardResp ShardSearchResponse
			if err := json.NewDecoder(resp.Body).Decode(&shardResp); err != nil {
				return
			}

			mu.Lock()
			aggregatedIDs = append(aggregatedIDs, shardResp.DocIDs...)
			mu.Unlock()
		}(url)

	}

	// 4. Wait for all requests to fininsh or abort via the timeout context
	wg.Wait()

	return aggregatedIDs
}
