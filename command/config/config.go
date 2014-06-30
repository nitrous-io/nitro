package config

import (
	"os"
	"path/filepath"
)

func APIServer() string {
	server := os.Getenv("NITRO_API_URL")

	if server == "" {
		server = "http://0.0.0.0:3000" // FIXME: set the production server?
	}

	return server
}

func TokenFilePath() string {
	// FIXME correct path for the token file
	return filepath.Join(os.Getenv("HOME"), ".nitrotoken")
}
