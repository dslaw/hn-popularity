package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewItem(t *testing.T) {
	id := ItemID(25260894)
	apiVersion := "v0"
	raw := `{
     "by": "User",
     "id": 25260894,
     "parent": 25260111,
     "text": "Indeed. RND works differently on the BBC.",
     "time": 1606783148,
     "type": "comment"
    }`

	item, err := NewItem(id, apiVersion, raw)

	if assert.Nil(t, err) {
		assert.Equal(t, id, item.ID)
		assert.Equal(t, int64(1606783148), item.CreatedAt)
		assert.NotEqual(t, *new(time.Time), item.ProcessedAt, "ProcessedAt is unset")
		assert.Equal(t, apiVersion, item.ApiVersion)
		assert.Equal(t, raw, item.RawItem)
	}
}
