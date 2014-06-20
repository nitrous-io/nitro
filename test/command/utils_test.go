package commandtest

import (
	"io/ioutil"
	"os"

	"github.com/nitrous-io/nitro/command/utils"

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

	Describe("LoadConfig", func() {
		Context("When config file exists", func() {
			BeforeEach(func() {
				sample_config := `
{
	"server": "https://www.example.com",
  "client_id": "client_id",
  "client_secret": "client_secret"
}
			`
				// assuming $HOME is set to /tmp
				ioutil.WriteFile("/tmp/.nitroconfig", []byte(sample_config), 0644)
			})

			AfterEach(func() {
				os.Remove("/tmp/.nitroconfig")
			})

			It("returns the read config", func() {
				config, err := utils.LoadConfig()
				Expect(err).NotTo(HaveOccurred())
				Expect(config.Server).To(Equal("https://www.example.com"))
			})
		})

		Context("When config file does not exist", func() {
			It("Returns an Error", func() {
				_, err := utils.LoadConfig()
				Expect(err).Should(HaveOccurred())
			})
		})
	})

	Describe("TokenFilePath", func() {
		It("returns a path to a dotfile in home directory", func() {
			Expect(utils.TokenFilePath()).To(Equal("/tmp/.nitrotoken"))
		})
	})
})
