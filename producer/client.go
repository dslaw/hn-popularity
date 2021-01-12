package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// HNClient implements an HTTP client for Hacker News.
type HNClient struct {
	client      *http.Client
	BaseURL     string
	ApiVersion  string
	RetryWait   time.Duration
	MaxAttempts int
}

func NewHNClient(
	httpTimeout time.Duration,
	baseURL string,
	apiVersion string,
	retryWait time.Duration,
	maxAttempts int,
) *HNClient {
	return &HNClient{
		client:      &http.Client{Timeout: httpTimeout},
		BaseURL:     baseURL,
		ApiVersion:  apiVersion,
		RetryWait:   retryWait,
		MaxAttempts: maxAttempts,
	}
}

func (client *HNClient) get(url string) (body string, err error) {
	var (
		rsp       *http.Response
		bodyBytes []byte
	)

	for attempt := 0; attempt < client.MaxAttempts; attempt++ {
		time.Sleep(time.Duration(attempt) * client.RetryWait)

		rsp, err = client.client.Get(url)
		if err != nil {
			// Protocol error.
			continue
		}
		defer rsp.Body.Close()

		bodyBytes, err = ioutil.ReadAll(rsp.Body)
		if err != nil {
			// Network error.
			continue
		}

		body = string(bodyBytes)

		switch {
		case rsp.StatusCode == 200 && body == "null":
			// XXX: The HN API will return 200 with a body of `null` for
			//      non-existent resources. This also occurs when the resource
			//      exists, but wasn't able to be retrieved for whatever reason.
			//      The latter case should be ephemeral, and can be resolved by
			//      retrying.
			continue
		case rsp.StatusCode == 200:
			return
		case rsp.StatusCode == 429:
			continue
		case rsp.StatusCode >= 500 && rsp.StatusCode < 600:
			continue
		default:
			break
		}
	}

	if err == nil {
		err = fmt.Errorf("http error: %d %s", rsp.StatusCode, body)
	}
	return
}

// MakeURL constructs the url for a resource's JSON endpoint.
func (client *HNClient) MakeURL(path ...string) string {
	parts := []string{client.BaseURL, client.ApiVersion}
	urlParts := append(parts, path...)
	return strings.Join(urlParts, "/") + ".json"
}

func (client *HNClient) GetMaxItemID() (id ItemID, err error) {
	url := client.MakeURL("maxitem")
	body, err := client.get(url)
	if err != nil {
		return
	}

	// ID is a JSON encoded integer, which can be parsed directly.
	id, err = NewItemIDFromString(body)
	return
}

func (client *HNClient) GetItem(id ItemID) (item *Item, err error) {
	url := client.MakeURL("item", ItemIDToString(id))
	body, err := client.get(url)
	if err != nil {
		return
	}

	item, err = NewItem(id, client.ApiVersion, body)
	return item, err
}

// LatestProducer is a producer that generates the latest items.
type LatestProducer struct {
	*HNClient
	maxItemID   ItemID
	itemID      ItemID
	initialized bool
}

func NewLatestProducer(client *HNClient) *LatestProducer {
	return &LatestProducer{client, ItemID(0), ItemID(0), false}
}

func (producer *LatestProducer) Next() (queuedItem *QueuedItem, err error) {
	// Check initialized.
	if !producer.initialized {
		var maxItemID ItemID
		maxItemID, err = producer.GetMaxItemID()
		if err != nil {
			return
		}

		producer.itemID = maxItemID
		producer.maxItemID = maxItemID
		producer.initialized = true
	}

	// Block until a new item is discovered.
	for {
		if producer.itemID <= producer.maxItemID {
			break
		}

		producer.maxItemID, err = producer.GetMaxItemID()
		if err != nil {
			return
		}
	}

	queuedItem = &QueuedItem{
		ID:        producer.itemID,
		CreatedAt: time.Now().UTC(),
	}
	producer.itemID++
	return
}

func (producer *LatestProducer) Name() string {
	return "new"
}

func (producer *LatestProducer) ProcessAt(queuedItem *QueuedItem) (time.Time, bool) {
	return time.Now().UTC(), false
}
