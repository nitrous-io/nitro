package config

import (
	"os"
	"path/filepath"
)

const authConfigFilename = ".nitrocfg" // assuming CLI will run on Unix based systems for now

func APIServerURL() string {
	server := os.Getenv("NITRO_API_URL")

	if server == "" {
		server = "http://0.0.0.0:3000" // FIXME: set the production server?
	}

	return server
}

func AuthConfigPath() string {
	authConfigPath := os.Getenv("NITRO_AUTH_CONFIG_PATH")
	if authConfigPath != "" {
		return authConfigPath
	}

	// Assumptions: CLI is running on Unix based systems
	//							User hasn't modified the default HOME env.
	return filepath.Join(os.Getenv("HOME"), authConfigFilename)
}
