package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueueProcessAt(t *testing.T) {
	t.Run("Expired item", func(t *testing.T) {
		now := time.Now().UTC()

		queuedItem := &QueuedItem{
			ID:        ItemID(1),
			CreatedAt: now.Add(-1 * time.Hour),
		}

		processAfter := 1 * time.Minute
		pq := NewPriorityQueue(nil, "queue", processAfter)

		_, expired := pq.ProcessAt(queuedItem)
		assert.True(t, expired, queuedItem.CreatedAt)
	})

	t.Run("Active item", func(t *testing.T) {
		now := time.Now().UTC()

		queuedItem := &QueuedItem{
			ID:        ItemID(1),
			CreatedAt: now.Add(1 * time.Hour),
		}

		processAfter := 1 * time.Hour
		pq := NewPriorityQueue(nil, "queue", processAfter)

		expected := now.Add(2 * time.Hour)

		actual, expired := pq.ProcessAt(queuedItem)
		assert.False(t, expired)
		assert.Equal(t, expected, actual)
	})

	t.Run("Active item with grace period", func(t *testing.T) {
		now := time.Now().UTC()

		queuedItem := &QueuedItem{
			ID:        ItemID(1),
			CreatedAt: now,
		}

		processAfter := 0 * time.Second
		pq := NewPriorityQueue(nil, "queue", processAfter)

		_, expired := pq.ProcessAt(queuedItem)
		assert.False(t, expired)
	})
}
