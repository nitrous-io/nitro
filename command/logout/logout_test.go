package logout_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/nitrous-io/nitro/auth"
	"github.com/nitrous-io/nitro/command"
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

		testConfigFile *os.File
	)

	BeforeEach(func() {
		ui = &cli.MockUi{}
		cmd = &LogoutCommand{UI: ui, Server: "www.example.com"}

		f, err := ioutil.TempFile("", "nitrocfg")
		Expect(err).To(BeNil())
		testConfigFile = f
		command.Env.ConfigPath = f.Name()
	})

	AfterEach(func() {
		testConfigFile.Close()
		os.Remove(testConfigFile.Name())
	})

	It("implements cli.Command", func() {
		var _ cli.Command = cmd
	})

	Context("when token exist in auth config file", func() {
		BeforeEach(func() {
			c := &auth.ConfigFile{Auths: make(map[string]auth.AuthConfig)}
			a := auth.AuthConfig{}
			a.Username = "foo"
			a.Token = "bar"
			c.Auths["www.example.com"] = a

			auth.SaveConfig(testConfigFile.Name(), c)
		})

		AfterEach(func() {
			err := os.Remove(testConfigFile.Name())
			if err != nil {
				panic(err)
			}
		})

		It("removes the token", func() {
			exitCode := cmd.Run(args)
			Expect(exitCode).To(Equal(0))

			c, err := auth.LoadConfig(testConfigFile.Name())
			if err != nil {
				Fail(err.Error())
			}
			Expect(c.Auths["www.example.com"].Username).To(BeEmpty())
			Expect(c.Auths["www.example.com"].Token).To(BeEmpty())
		})
	})
})
