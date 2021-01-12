package main

import (
	"encoding/json"
	"strconv"
	"time"
)

type ItemID int64

func NewItemIDFromString(s string) (id ItemID, err error) {
	idInt, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return
	}
	id = ItemID(idInt)
	return
}

func ItemIDToString(id ItemID) string {
	return strconv.FormatInt(int64(id), 10)
}

// QueuedItem represents an item to be processed.
type QueuedItem struct {
	ID        ItemID
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
