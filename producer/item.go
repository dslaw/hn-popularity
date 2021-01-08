package main

import (
	"encoding/json"
	"time"
)

type ItemID int64

// QueuedItem represents an item to be processed.
type QueuedItem struct {
	ID        ItemID
	FromQueue bool
	CreatedAt time.Time
}

// Item represents a processed item, wrapping a raw HN item and its metadata.
type Item struct {
	ID          ItemID
	CreatedAt   int64
	ProcessedAt time.Time
	ApiVersion  string
	RawItem     string
}

func NewItem(id ItemID, apiVersion string, raw string) (item *Item, err error) {
	// Deserialize so `CreatedAt` can be set as metadata.
	obj := make(map[string]interface{})
	err = json.Unmarshal([]byte(raw), &obj)
	if err != nil {
		return
	}
	createdAt := obj["time"].(float64)

	item = &Item{
		ID:          id,
		CreatedAt:   int64(createdAt),
		ProcessedAt: time.Now().UTC(),
		ApiVersion:  apiVersion,
		RawItem:     raw,
	}
	return
}
