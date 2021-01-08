package main

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// PriorityQueue is a producer that generates items from a priority queue.
// Items in the queue are ordered by their creation time, with earlier created
// items taking higher priority.
type PriorityQueue struct {
	client       *redis.Client
	ctx          context.Context
	QueueName    string
	ExpiryWindow time.Duration
}

func NewPriorityQueue(client *redis.Client, name string) *PriorityQueue {
	return &PriorityQueue{
		client:       client,
		ctx:          context.Background(),
		QueueName:    name,
		ExpiryWindow: 0 * time.Second, // Indefinite.
	}
}

func (pq *PriorityQueue) PushBack(item *Item) error {
	z := &redis.Z{
		Score:  float64(item.CreatedAt),
		Member: int64(item.ID),
	}
	err := pq.client.ZAdd(pq.ctx, pq.QueueName, z).Err()
	return err
}

func zToQueuedItem(z redis.ZWithKey) (queuedItem *QueuedItem, err error) {
	idStr := z.Member.(string)
	id, err := NewItemIDFromString(idStr)
	if err != nil {
		return
	}
	createdAt := int64(z.Score)

	queuedItem = &QueuedItem{
		ID:        ItemID(id),
		FromQueue: true,
		CreatedAt: time.Unix(createdAt, 0),
	}
	return
}

func (pq *PriorityQueue) Next() (queuedItem *QueuedItem, err error) {
	// Block until a new item is available.
	timeout := 0 * time.Second // Indefinite.
	z, err := pq.client.BZPopMin(pq.ctx, timeout, pq.QueueName).Result()
	if err != nil {
		return
	}

	if z != nil {
		queuedItem, err = zToQueuedItem(*z)
	}
	return
}

func (pq *PriorityQueue) Name() string {
	return pq.QueueName
}
