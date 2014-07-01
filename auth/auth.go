package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/bgentry/go-netrc/netrc"
	"github.com/nitrous-io/nitro/command/config"
)

type NetRc struct {
	netrc.Netrc
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

func loadNetRc() (*NetRc, error) {
	onrc, err := netrc.ParseFile(config.NetRcPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &NetRc{}, nil
		}
		return nil, err
	}

	return &NetRc{*onrc}, nil
}

func SaveToken(host, user, token string) error {
	nrc, err := loadNetRc()
	if err != nil {
		return err
	}

	m := nrc.FindMachine(host)
	if m == nil || m.IsDefault() {
		m = nrc.NewMachine(host, user, token, "")
	}
	m.UpdateLogin(user)
	m.UpdatePassword(token)

	body, err := nrc.MarshalText()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(config.NetRcPath(), body, 0600)
}

func RemoveToken(host string) error {
	nrc, err := loadNetRc()
	if err != nil {
		return err
	}

	nrc.RemoveMachine(host)

	body, err := nrc.MarshalText()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(config.NetRcPath(), body, 0600)
}
