package nitrousapi

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"

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
	Domain         string `json:"domain"`
	CA             string `json:"ca"`
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

	hostEndpoint := fmt.Sprintf("%s/api/v0/hosts/%d", server, cr.Container.HostID)
	resp, err = client.GetAuthed(hostEndpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to find host for container %s", slug)
	}

	type hostResp struct {
		Host *HostAPI `json:"host"`
	}
	h := &hostResp{}
	if err := json.NewDecoder(resp.Body).Decode(h); err != nil {
		return nil, err
	}
	cr.Container.Host = h.Host

	return cr.Container, nil
}

// TLSConfig returns a tls configuration generated from api response that is
// valid for establishing HTTPS connection with the dedicated host
func TLSConfig(h *HostAPI) (*tls.Config, error) {
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM([]byte(h.CA))
	tlsConfig := &tls.Config{
		ServerName: h.Domain,
		RootCAs:    certPool,
	}
	return tlsConfig, nil
}
