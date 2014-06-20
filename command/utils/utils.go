package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type (
	config struct {
		Server       string `json:"server"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
)

func LoadConfig() (*config, error) {
	config := &config{}

	// FIXME path of config file
	// currently assumes ~/.nitro_config
	configData, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".nitroconfig"))
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(configData, config)
	if err != nil {
		err = errors.New(fmt.Sprintf("invalid config data %v", string(configData)))
	}

	return config, err
}

func TokenFilePath() string {
	// FIXME correct path for the token file
	return filepath.Join(os.Getenv("HOME"), ".nitrotoken")
}
