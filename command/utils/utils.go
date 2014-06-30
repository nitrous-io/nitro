package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Config struct {
	Server       string `json:"server"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	// FIXME path of config file
	// currently assumes ~/.nitroconfig
	configData, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".nitroconfig"))
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(configData, config)
	if err != nil {
		err = fmt.Errorf("invalid config data %v", string(configData))
	}

	return config, err
}

func TokenFilePath() string {
	// FIXME correct path for the token file
	return filepath.Join(os.Getenv("HOME"), ".nitrotoken")
}
