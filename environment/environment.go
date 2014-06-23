package environment

import (
	"os"
	"path"
	"strconv"
)

type Environment struct {
	BaseDir    string
	ConfigFile string // path to config file
	SSLVerify  bool
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

	noSSLVerify, _ := strconv.ParseBool(os.Getenv("NO_SSL_VERIFY"))
	e.SSLVerify = !noSSLVerify

	return e
}
