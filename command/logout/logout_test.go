package logout_test

import (
	"os"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/auth"
	. "github.com/nitrous-io/nitro/command/logout"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLogout(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "logout")
}

var _ = Describe("LogoutCommand", func() {
	var (
		cmd  *LogoutCommand
		ui   *cli.MockUi
		args []string

		currentAuthConfigPath string
		testAuthConfigPath    string
	)

	BeforeEach(func() {
		ui = &cli.MockUi{}
		cmd = &LogoutCommand{UI: ui, Server: "www.example.com"}

		testAuthConfigPath = "/tmp/authcfg"
		currentAuthConfigPath = os.Getenv("NITRO_AUTH_CONFIG_PATH")
		os.Setenv("NITRO_AUTH_CONFIG_PATH", testAuthConfigPath)
	})

	AfterEach(func() {
		os.Setenv("NITRO_AUTH_CONFIG_PATH", currentAuthConfigPath)
	})

	It("implements cli.Command", func() {
		var _ cli.Command = cmd
	})

	Context("when token exist in auth config file", func() {
		BeforeEach(func() {
			c := &auth.ConfigFile{Configs: make(map[string]auth.AuthConfig)}
			a := auth.AuthConfig{}
			a.Username = "foo"
			a.Token = "bar"
			c.Configs["www.example.com"] = a

			auth.SaveConfig(testAuthConfigPath, c)
		})

		AfterEach(func() {
			err := os.Remove(testAuthConfigPath)
			if err != nil {
				panic(err)
			}
		})

		It("removes the token", func() {
			exitCode := cmd.Run(args)
			Expect(exitCode).To(Equal(0))

			c, err := auth.LoadConfig(testAuthConfigPath)
			if err != nil {
				Fail(err.Error())
			}
			Expect(c.Configs["www.example.com"].Username).To(BeEmpty())
			Expect(c.Configs["www.example.com"].Token).To(BeEmpty())
		})
	})
})
