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
	Describe("APIServer", func() {
		Context("when NITRO_API_URL env variable is empty", func() {
			It("returns the default server host", func() {
				Expect(config.APIServer()).To(Equal("http://0.0.0.0:3000"))
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
				Expect(config.APIServer()).To(Equal("https://www.example.com"))
			})
		})
	})

	Describe("NetRcPath", func() {
		Context("When NETRC_PATH env variable is set", func() {
			var currentNetrcPath string

			BeforeEach(func() {
				currentNETRCPath = os.Getenv("NETRC_PATH")
				os.Setenv("NETRC_PATH", "/fake/fs/.netrc")
			})

			AfterEach(func() {
				os.Setenv("NETRC_PATH", currentNetrcPath)
			})

			It("returns the path set in NETRC_PATH env variable", func() {
				Expect(config.NetrcPath()).To(Equal("/fake/fs/.netrc"))
			})
		})

		Context("When NETRC_PATH env variable is not set", func() {
			var currentNetrcPath string

			BeforeEach(func() {
				currentNETRCPath = os.Getenv("NETRC_PATH")
				os.Setenv("NETRC_PATH", "")
			})

			AfterEach(func() {
				os.Setenv("NETRC_PATH", currentNetrcPath)
			})

			It("returns the path to .netrc in user's home directory", func() {
				Expect(config.NetrcPath()).To(MatchRegexp(`^\/home\/*\/.netrc$`))
			})
		})
	})
})
