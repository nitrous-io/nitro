package nitrousapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/nitrous-io/nitrous-common/httpclient"
)

type ContainerAPI struct {
	ID        int64  `json:"id"`
	CID       string `json:"cid"`
	UserID    int64  `json:"user_id"`
	Image     string `json:"image"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
	State     string `json:"state"`

	HostID int64 `json:"host_id"`
	Host   *HostAPI
}

type HostAPI struct {
	ID             int64  `json:"id"`
	Slug           string `json:"slug"`
	IPAddress      string `json:"ip_address"`
	FreighterdPort int64  `json:"freighterd_port"`
	Hostname       string `json:"hostname"`
	ProxyHostname  string `json:"proxy_hostname"`
}

type hostResp struct {
	Host *HostAPI `json:"host"`
}

// GetHost returns the host api response when we give it either the host's slug
// or the host id ( string representation of its int64 value )
func GetHost(client *httpclient.Client, server, slugOrID string) (*HostAPI, error) {
	hostEndpoint := fmt.Sprintf("%s/api/v0/hosts/%s", server, slugOrID)
	resp, err := client.GetAuthed(hostEndpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to find host %s", slugOrID)
	}

	h := &hostResp{}
	if err := json.NewDecoder(resp.Body).Decode(h); err != nil {
		return nil, err
	}
	return h.Host, nil
}

func GetContainer(client *httpclient.Client, server, slug string) (*ContainerAPI, error) {
	// get container data
	endpoint := fmt.Sprintf("%s/api/v0/containers/%s", server, slug)
	resp, err := client.GetAuthed(endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to find container %s", slug)
	}
	type containerResp struct {
		Container *ContainerAPI `json:"container"`
	}

	cr := &containerResp{}
	if err := json.NewDecoder(resp.Body).Decode(cr); err != nil {
		return nil, err
	}

	// container embeds its host information so we want to do another request
	// to the server to get it
	h, err := GetHost(client, server, strconv.FormatInt(cr.Container.HostID, 10))
	if err != nil {
		return nil, err
	}
	cr.Container.Host = h

	return cr.Container, nil
}
