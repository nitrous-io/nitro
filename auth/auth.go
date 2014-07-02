package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/nitrous-io/nitro/command/config"
)

type AuthConfig struct {
	Username string `json:"username"`
	Token    string `json:"auth"`
}

type ConfigFile struct {
	Configs map[string]AuthConfig `json:"configs"`
}

func FetchToken(server, username, password string) (string, error) {
	p := url.Values{}
	p.Set("grant_type", "password")
	p.Set("username", username)
	p.Set("password", password)

	url := fmt.Sprintf("%s/oauth/token", server)
	response, err := http.PostForm(url, p)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var message struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		AccessToken      string `json:"access_token"`
	}

	err = json.NewDecoder(response.Body).Decode(&message)
	if err != nil {
		return "", err
	}

	switch response.StatusCode {
	case 401:
		return "", fmt.Errorf(message.ErrorDescription)
	case 200:
		return message.AccessToken, nil
	default:
		return "", fmt.Errorf("failed to authenticate %v", response.Status)
	}
}

func LoadConfig(path string) (*ConfigFile, error) {
	configFile := ConfigFile{Configs: make(map[string]AuthConfig)}
	fd, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &configFile, nil
		}
		return nil, err
	}
	defer fd.Close()

	b, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &configFile.Configs)
	if err != nil {
		return nil, err
	}

	return &configFile, nil
}

func SaveConfig(path string, configFile *ConfigFile) error {
	b, err := json.Marshal(&configFile.Configs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0600)
}

func SaveToken(host, username, token string) error {
	configPath := config.AuthConfigPath()
	c, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	authConfig := AuthConfig{}
	authConfig.Username = username
	authConfig.Token = token
	c.Configs[host] = authConfig

	return SaveConfig(configPath, c)
}

func RemoveToken(host string) error {
	configPath := config.AuthConfigPath()
	c, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	delete(c.Configs, host)

	return SaveConfig(configPath, c)
}
