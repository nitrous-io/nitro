package config_test

import (
	"os"
	"testing"

	"github.com/nitrous-io/nitro/command/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestContext(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "config")
}

var _ = Describe("Config", func() {
	Describe("APIServerURL", func() {
		Context("when NITRO_API_URL env variable is empty", func() {
			It("returns the default server url", func() {
				Expect(config.APIServerURL()).To(Equal("http://0.0.0.0:3000"))
			})
		})

		Context("when NITRO_API_URL env variable is set", func() {
			BeforeEach(func() {
				os.Setenv("NITRO_API_URL", "https://www.example.com")
			})

			AfterEach(func() {
				os.Setenv("NITRO_API_URL", "")
			})

			It("returns server set in the env variable", func() {
				Expect(config.APIServerURL()).To(Equal("https://www.example.com"))
			})
		})
	})

	Describe("AuthConfigPath", func() {
		Context("When NITRO_AUTH_CONFIG_PATH env variable is set", func() {
			var currentAuthConfigPath string

			BeforeEach(func() {
				currentAuthConfigPath = os.Getenv("NITRO_AUTH_CONFIG_PATH")
				os.Setenv("NITRO_AUTH_CONFIG_PATH", "/fake/fs/.netrc")
			})

			AfterEach(func() {
				os.Setenv("NITRO_AUTH_CONFIG_PATH", currentAuthConfigPath)
			})

			It("returns the path set in NITRO_AUTH_CONFIG_PATH env variable", func() {
				Expect(config.AuthConfigPath()).To(Equal("/fake/fs/.netrc"))
			})
		})

		Context("When NITRO_AUTH_CONFIG_PATH env variable is not set", func() {
			var currentAuthConfigPath string

			BeforeEach(func() {
				currentAuthConfigPath = os.Getenv("NITRO_AUTH_CONFIG_PATH")
				os.Setenv("NITRO_AUTH_CONFIG_PATH", "")
			})

			AfterEach(func() {
				os.Setenv("NITRO_AUTH_CONFIG_PATH", currentAuthConfigPath)
			})

			It("returns the path to .nitrocfg in user's home directory", func() {
				Expect(config.AuthConfigPath()).To(MatchRegexp(`^/home/.*/.nitrocfg$`))
			})
		})
	})
})
