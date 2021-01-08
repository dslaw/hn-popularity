package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func NewTestClient(f RoundTripFunc) *HNClient {
	httpClient := &http.Client{
		Transport: RoundTripFunc(f),
	}
	return &HNClient{
		client:      httpClient,
		BaseURL:     "https://hacker-news.firebaseio.com",
		ApiVersion:  "v0",
		MaxAttempts: 2,
		RetryWait:   0 * time.Second,
	}
}

func TestHNClientGet(t *testing.T) {
	t.Run("Returns body when 200", func(t *testing.T) {
		body := "Ok"
		client := NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			}
		})

		actual, err := client.get(client.BaseURL)
		if assert.Nil(t, err) {
			assert.Equal(t, body, actual)
		}
	})

	t.Run("Retries when 200 with null body", func(t *testing.T) {
		attempt := 1
		bodyOK := "Ok"
		bodyWrong := "null"

		client := NewTestClient(func(req *http.Request) *http.Response {
			body := bodyWrong
			if attempt > 1 {
				body = bodyOK
			}
			attempt++

			return &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			}
		})

		actual, err := client.get(client.BaseURL)
		if assert.Nil(t, err) {
			assert.Equal(t, bodyOK, actual)
		}
	})

	t.Run("Returns error when 4xx", func(t *testing.T) {
		client := NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 400,
				Body:       ioutil.NopCloser(bytes.NewBufferString("Client error")),
			}
		})

		_, err := client.get(client.BaseURL)
		assert.NotNil(t, err)
	})

	t.Run("Retries when 5xx", func(t *testing.T) {
		attempt := 1
		bodyOk := "Ok"

		client := NewTestClient(func(req *http.Request) *http.Response {
			statusCode := 500
			body := "Server error"

			if attempt > 1 {
				statusCode = 200
				body = bodyOk
			}
			attempt++

			return &http.Response{
				StatusCode: statusCode,
				Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
			}
		})

		actual, err := client.get(client.BaseURL)
		if assert.Nil(t, err) {
			assert.Equal(t, bodyOk, actual)
		}
	})

	t.Run("Returns error when retries exceeded", func(t *testing.T) {
		client := NewTestClient(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 500,
				Body:       ioutil.NopCloser(bytes.NewBufferString("Client error")),
			}
		})

		_, err := client.get(client.BaseURL)
		assert.NotNil(t, err)
	})
}

func TestHNClientMakeURL(t *testing.T) {
	client := &HNClient{
		BaseURL:    "http://localhost.com",
		ApiVersion: "v1",
	}
	expected := "http://localhost.com/v1/path/1.json"
	actual := client.MakeURL("path", "1")
	assert.Equal(t, expected, actual)
}

func TestHNClientGetMaxItemID(t *testing.T) {
	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString("2343")),
		}
	})

	itemID, err := client.GetMaxItemID()
	if assert.Nil(t, err) {
		assert.Equal(t, ItemID(2343), itemID)
	}
}

func TestHNClientGetItem(t *testing.T) {
	body := `{
     "by": "User",
     "id": 25260894,
     "parent": 25260111,
     "text": "Indeed. RND works differently on the BBC.",
     "time": 1606783148,
     "type": "comment"
    }`

	client := NewTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		}
	})

	itemID := ItemID(25260894)
	expected, _ := NewItem(itemID, client.ApiVersion, body)
	actual, err := client.GetItem(itemID)

	if assert.Nil(t, err) {
		assert.Equal(t, expected.ID, actual.ID)
		assert.NotEqual(t, new(int64), actual.CreatedAt)
		assert.NotEqual(t, new(time.Time), actual.ProcessedAt)
		assert.Equal(t, expected.ApiVersion, actual.ApiVersion)
		assert.Equal(t, expected.RawItem, actual.RawItem)
	}
}

func TestNewLatestProducer(t *testing.T) {
	producer := NewLatestProducer(nil)
	assert.False(t, producer.initialized)
}

func TestLatestProducerNext(t *testing.T) {
	itemID := 1
	client := NewTestClient(func(req *http.Request) *http.Response {
		body := strconv.Itoa(itemID)
		itemID++
		return &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		}
	})

	producer := NewLatestProducer(client)

	// First call initializes the producer and returns next.
	actual, err := producer.Next()
	require.Nil(t, err)
	assert.Equal(t, ItemID(1), actual.ID)
	assert.NotEqual(t, actual.CreatedAt, new(time.Time))

	// Internal state is updated.
	assert.Equal(t, ItemID(2), producer.itemID)
	assert.Equal(t, ItemID(1), producer.maxItemID)

	// Non-first call returns next.
	actual, err = producer.Next()
	require.Nil(t, err)
	assert.Equal(t, ItemID(2), actual.ID)
	assert.NotEqual(t, actual.CreatedAt, new(time.Time))

	// Internal state is updated.
	assert.Equal(t, ItemID(3), producer.itemID)
	assert.Equal(t, ItemID(2), producer.maxItemID)
}
