package config

import (
	"os"
	"path/filepath"
)

const netrcFilename = ".netrc" // assuming CLI will run on Unix based systems for now

func APIServer() string {
	server := os.Getenv("NITRO_API_URL")

	if server == "" {
		server = "http://0.0.0.0:3000" // FIXME: set the production server?
	}

	return server
}

func NetRcPath() string {
	netRcPath := os.Getenv("NETRC_PATH")
	if netRcPath != "" {
		return netRcPath
	}

	// Assumptions: CLI is running on Unix based systems
	//							User hasn't modified the default HOME env.
	return filepath.Join(os.Getenv("HOME"), netrcFilename)
}
