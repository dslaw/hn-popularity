package main

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigPath := tmpDir + "/" + "test_config.yml"
	configData := `
    client:
      http_timeout: 1s
      api_version: v0
      base_url: "http://localhost"
      retry_wait: 1s
      max_attempts: 1

    channels:
      queue_1:
        process_after: 0s
        next: queue_2
      queue_2:
        process_after: 1s
        next:
    `

	ioutil.WriteFile(testConfigPath, []byte(configData), 0644)

	config, err := NewConfigFromFile(testConfigPath)

	require.Nil(t, err)
	require.NotNil(t, config)

	assert.Equal(t, 1*time.Second, config.Client.HTTPTimeout)
	assert.Equal(t, "v0", config.Client.APIVersion)
	assert.Equal(t, "http://localhost", config.Client.BaseURL)
	assert.Equal(t, 1*time.Second, config.Client.RetryWait)
	assert.Equal(t, 1, config.Client.MaxAttempts)

	assert.Equal(t, 0*time.Second, config.Channels["queue_1"].ProcessAfter)
	assert.NotNil(t, config.Channels["queue_1"].Next)
	assert.Equal(t, "queue_2", *config.Channels["queue_1"].Next)

	assert.Equal(t, 1*time.Second, config.Channels["queue_2"].ProcessAfter)
	assert.Nil(t, config.Channels["queue_2"].Next)
}
