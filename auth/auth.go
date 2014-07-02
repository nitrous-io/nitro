package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/nitrous-io/nitro/command"
)

type AuthConfig struct {
	Username string `json:"username"`
	Token    string `json:"auth"`
}

type ConfigFile struct {
	Auths map[string]AuthConfig `json:"auths"`
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
	configFile := ConfigFile{Auths: make(map[string]AuthConfig)}
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
	if len(b) == 0 {
		return &configFile, nil
	}

	err = json.Unmarshal(b, &configFile.Auths)
	if err != nil {
		return nil, err
	}

	return &configFile, nil
}

func SaveConfig(path string, configFile *ConfigFile) error {
	b, err := json.Marshal(&configFile.Auths)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0600)
}

func SaveToken(host, username, token string) error {
	c, err := LoadConfig(command.Env.ConfigPath)
	if err != nil {
		return err
	}

	authConfig := AuthConfig{}
	authConfig.Username = username
	authConfig.Token = token
	c.Auths[host] = authConfig

	return SaveConfig(command.Env.ConfigPath, c)
}

func RemoveToken(host string) error {
	c, err := LoadConfig(command.Env.ConfigPath)
	if err != nil {
		return err
	}

	delete(c.Auths, host)

	return SaveConfig(command.Env.ConfigPath, c)
}
