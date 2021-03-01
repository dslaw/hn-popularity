package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/mattn/go-sqlite3"
)

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

	config, err := NewConfigFromFile("/app/config.yml")
	if err != nil {
		log.Panicf("Error reading 'config.yml': %s", err)
	}

	inQueueConfig, exists := config.Channels[inQueueName]
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

	client := NewHNClient(
		config.Client.HTTPTimeout,
		config.Client.BaseURL,
		config.Client.APIVersion,
		config.Client.RetryWait,
		config.Client.MaxAttempts,
	)

	var (
		producer Producer
		outQueue *PriorityQueue
	)
	if inQueueName == "new" {
		producer = NewLatestProducer(client)
	} else {
		producer = NewPriorityQueue(rdb, inQueueName, inQueueConfig.ProcessAfter)
	}
	if inQueueConfig.Next != nil {
		// `processAfter` argument isn't used for the output queue.
		outProcessAfter := 0 * time.Second
		outQueue = NewPriorityQueue(rdb, *inQueueConfig.Next, outProcessAfter)
	}

	Process(producer, client, outQueue, store)
}
