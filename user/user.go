package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nitrous-io/nitro/command/utils"
)

func GetAccessToken(username, password string, config *utils.Config) (string, error) {
	p := url.Values{}
	p.Set("grant_type", "password")
	p.Set("username", username)
	p.Set("password", password)
	p.Set("client_id", config.ClientID)
	p.Set("client_secret", config.ClientSecret)

	url := fmt.Sprintf("%s/oauth/token", config.Server)
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
