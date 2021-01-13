package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/mattn/go-sqlite3"
)

const (
	httpTimeout = 15 * time.Second
	apiVersion  = "v0"
	baseURL     = "https://hacker-news.firebaseio.com"
	retryWait   = 1 * time.Second
	maxAttempts = 5
)

func GetQueueConfig(inQueueName string) (*string, time.Duration, bool) {
	var (
		outQueueName *string
		processAfter time.Duration
		found        bool
	)

	queues := []struct {
		Queue        string
		ProcessAfter time.Duration
	}{
		{
			Queue:        "new",
			ProcessAfter: 0 * time.Second,
		}, {
			Queue:        "queue:15m",
			ProcessAfter: 15 * time.Minute,
		}, {
			Queue:        "queue:30m",
			ProcessAfter: 30 * time.Minute,
		}, {
			Queue:        "queue:1h",
			ProcessAfter: 1 * time.Hour,
		},
	}

	for idx, queue := range queues {
		if queue.Queue == inQueueName {
			found = true
			processAfter = queue.ProcessAfter
			nextIdx := idx + 1
			if nextIdx < len(queues) {
				outQueueName = &queues[nextIdx].Queue
			}
		}
	}

	return outQueueName, processAfter, found
}

type ItemRepo interface {
	Save(*Item, string) error
}

type ItemStore struct {
	insertStmt *sql.Stmt
}

func NewItemStore(db *sql.DB) (store *ItemStore, err error) {
	insertStmt, err := db.Prepare(`
    insert into items (id, processed_at, api_version, channel, item)
    values (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return
	}

	store = &ItemStore{insertStmt: insertStmt}
	return
}

func (store *ItemStore) Save(item *Item, channel string) error {
	_, err := store.insertStmt.Exec(
		item.ID,
		item.ProcessedAt.UTC().Format(time.RFC3339),
		item.ApiVersion,
		channel,
		item.RawItem,
	)
	return err
}

// Process retrieves and stores items from HN.
func Process(producer Producer, client *HNClient, outQueue *PriorityQueue, repo ItemRepo) {
	inChannel := producer.Name()

	for {
		queuedItem, err := producer.Next()
		if err != nil {
			// TODO: Panic?
			log.Printf("Unable to get next item: %s", err)
			continue
		}

		processAt, expired := producer.ProcessAt(queuedItem)
		time.Sleep(time.Until(processAt))
		if expired {
			log.Printf("Expired item %d", queuedItem.ID)
			continue
		}

		item, err := client.GetItem(queuedItem.ID)
		if err != nil {
			log.Printf("Unable to get item %d: %s", queuedItem.ID, err)
			continue
		}

		if err := repo.Save(item, inChannel); err != nil {
			log.Printf("Unable to save item %d: %s", queuedItem.ID, err)
			continue
		}

		if outQueue == nil {
			continue
		}

		if err := outQueue.PushBack(item); err != nil {
			log.Printf("Unable to push item %d: %s", queuedItem.ID, err)
			continue
		}
	}
}

func GetEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Panic("Env var not found")
	}
	return value
}

func main() {
	inQueueName := GetEnv("PRODUCER_IN_QUEUE")
	databaseURL := GetEnv("PRODUCER_DATABASE_URL")
	queueURL := GetEnv("PRODUCER_QUEUE_URL")

	outQueueName, processAfter, exists := GetQueueConfig(inQueueName)
	if !exists {
		log.Panicf("No such input queue: %s", inQueueName)
	}

	db, err := sql.Open("sqlite3", databaseURL)
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()
	store, err := NewItemStore(db)
	if err != nil {
		log.Panic(err)
	}

	opt, err := redis.ParseURL(queueURL)
	if err != nil {
		log.Panic(err)
	}
	rdb := redis.NewClient(opt)

	var (
		producer Producer
		outQueue *PriorityQueue
	)

	client := NewHNClient(httpTimeout, baseURL, apiVersion, retryWait, maxAttempts)
	if inQueueName == "new" {
		producer = NewLatestProducer(client)
	} else {
		producer = NewPriorityQueue(rdb, inQueueName, processAfter)
	}
	if outQueueName != nil {
		// `processAfter` argument isn't used for the output queue.
		outProcessAfter := 0 * time.Second
		outQueue = NewPriorityQueue(rdb, *outQueueName, outProcessAfter)
	}

	Process(producer, client, outQueue, store)
}
