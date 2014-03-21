package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"gopkg.in/v1/yaml"
)

type Config struct {
	OAuth struct {
		Key    string
		Secret string
	}
}

func ReadConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	// validate that required fields were set
	if config.OAuth.Key == "" {
		return nil, errors.New("missing field: oauth key")
	}
	if config.OAuth.Secret == "" {
		return nil, errors.New("missing field: oauth secret")
	}

	return &config, nil
}

func LoadConfig() (*Config, error) {
	home := os.Getenv("HOME")
	if home == "" {
		return nil, errors.New("HOME is not set in environment")
	}
	path := filepath.Join(home, ".config", "twackup", "oauth.yaml")
	config, err := ReadConfig(path)
	return config, err
}

func GetCredentials(config *Config) (client *twittergo.Client) {
	oc := &oauth1a.ClientConfig{
		ConsumerKey:    config.OAuth.Key,
		ConsumerSecret: config.OAuth.Secret,
	}
	return twittergo.NewClient(oc, nil)
}
