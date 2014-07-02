package command_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nitrous-io/nitro/command"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvironment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "environment")
}

var _ = Describe("environment", func() {
	var originalEnvironment *command.Environment

	BeforeEach(func() {
		originalEnvironment = command.ConfigureEnvironment()
	})

	AfterEach(func() {
		os.Setenv("NITRO_CONFIG_PATH", originalEnvironment.ConfigPath)
		os.Setenv("NITRO_API_URL", originalEnvironment.ApiURL)
	})

	Context("ApiURL", func() {
		It("sets from environment", func() {
			os.Setenv("NITRO_API_URL", "http://example.com")
			Expect(command.ConfigureEnvironment().ApiURL).To(Equal("http://example.com"))
		})

		It("defaults to ~/.nitrocfg", func() {
			os.Setenv("NITRO_API_URL", "")
			Expect(command.ConfigureEnvironment().ApiURL).To(Equal("http://127.0.0.1:3000"))
		})
	})

	Context("ConfigPath", func() {
		It("sets from environment", func() {
			os.Setenv("NITRO_CONFIG_PATH", "/tmp/myconf")
			Expect(command.ConfigureEnvironment().ConfigPath).To(Equal("/tmp/myconf"))
		})

		It("defaults to ~/.nitrocfg", func() {
			os.Setenv("NITRO_CONFIG_PATH", "")
			Expect(command.ConfigureEnvironment().ConfigPath).To(Equal(filepath.Join(os.Getenv("HOME"), ".nitrocfg")))
		})
	})
})
