package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

type ChannelConfig struct {
	ProcessAfter time.Duration `yaml:"process_after"`
	Next         *string       `yaml:"next"`
}

type Config struct {
	Client struct {
		HTTPTimeout time.Duration `yaml:"http_timeout"`
		APIVersion  string        `yaml:"api_version"`
		BaseURL     string        `yaml:"base_url"`
		RetryWait   time.Duration `yaml:"retry_wait"`
		MaxAttempts int           `yaml:"max_attempts"`
	} `yaml:"client"`
	Channels map[string]ChannelConfig `yaml:"channels"`
}

func NewConfigFromFile(path string) (config *Config, err error) {
	src, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	err = yaml.UnmarshalStrict(src, &config)
	return
}
