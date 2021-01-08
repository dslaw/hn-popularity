package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// HNClient implements an HTTP client for Hacker News.
type HNClient struct {
	client      *http.Client
	BaseURL     string
	ApiVersion  string
	RetryWait   time.Duration
	MaxAttempts int
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
