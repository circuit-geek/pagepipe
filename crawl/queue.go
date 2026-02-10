// Package crawl — BFS queue with deduplication.
// Maintains a visited set to avoid processing the same URL twice.
package crawl

// Queue is a BFS queue with URL deduplication.
type Queue struct {
	items   []string
	visited map[string]bool
	idx     int // current read position
}

// NewQueue creates an empty Queue.
func NewQueue() *Queue {
	return &Queue{
		visited: make(map[string]bool),
	}
}

// Add enqueues a URL if it hasn't been seen before.
func (q *Queue) Add(url string) {
	if q.visited[url] {
		return
	}
	q.visited[url] = true
	q.items = append(q.items, url)
}

// HasNext returns true if there are unprocessed URLs.
func (q *Queue) HasNext() bool {
	return q.idx < len(q.items)
}

// Next returns the next unprocessed URL and advances the pointer.
func (q *Queue) Next() string {
	url := q.items[q.idx]
	q.idx++
	return url
}

// Visited returns the total number of unique URLs seen.
func (q *Queue) Visited() int {
	return len(q.visited)
}

// All returns all discovered URLs (in BFS order).
func (q *Queue) All() []string {
	return q.items
}
