package main

// Producer defines a blocking iterator that generates new items to be
// processed.
type Producer interface {
	Next() (*QueuedItem, error)
	Name() string
}
