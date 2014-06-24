package environment

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

type Environment struct {
	BaseDir    string
	ConfigFile string // path to config file
	Config
}

type Config struct {
	Server       string `json:"server"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func BaseEnvironmentWithConfig() (Environment, error) {
	e := BaseEnvironment()
	err := e.PopulateConfig()
	if err != nil {
		return Environment{}, err
	}
	return e, nil
}

// BaseEnvironment is what nitro actually uses: parses env vars, config file,
// etc
func BaseEnvironment() Environment {
	e := Environment{}

	// TODO: follow XDG basedir spec
	// http://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html
	baseDir := os.Getenv("NITRO_DIR")
	if baseDir == "" {
		baseDir = path.Join(os.Getenv("HOME"), ".nitro")
	}
	e.BaseDir = baseDir
	e.ConfigFile = path.Join(baseDir, "config")

	return e
}

func (e *Environment) PopulateConfig() error {
	configJson, err := loadConfig(e.ConfigFile)
	if err != nil {
		return err
	}
	config, err := ParseConfig(configJson)
	if err != nil {
		return err
	}
	e.Config = *config
	return nil
}

func ParseConfig(data []byte) (*Config, error) {
	config := &Config{}

	err := json.Unmarshal(data, config)
	if err != nil {
		err = fmt.Errorf("invalid config data: %s", data)
	}

	return config, err
}

func loadConfig(path string) ([]byte, error) {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return []byte{}, fmt.Errorf("could not find config file at %s", path)
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return []byte{}, err
	}
	if len(data) == 0 {
		return []byte{}, fmt.Errorf("config file is empty")
	}
	return data, nil
}
