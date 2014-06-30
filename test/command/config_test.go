package commandtest

import (
	"os"

	"github.com/nitrous-io/nitro/command/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils", func() {
	var currentHome string
	BeforeEach(func() {
		currentHome = os.Getenv("HOME")
		os.Setenv("HOME", "/tmp")
	})

	AfterEach(func() {
		os.Setenv("HOME", currentHome)
	})

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

	Describe("TokenFilePath", func() {
		It("returns a path to a dotfile in home directory", func() {
			Expect(config.TokenFilePath()).To(Equal("/tmp/.nitrotoken"))
		})
	})
})
