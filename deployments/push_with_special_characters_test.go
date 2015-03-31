package deployments

import (
//	"fmt"
	"io/ioutil"
	"os"
//	"path"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
//	  "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = Describe("Deploy Apps", func() {
	var (
		appName		string
		BuildpackName	string

		appPath string

	)


	BeforeEach(func() {
		AsUser(context.AdminUserContext(), func() {
			BuildpackName = RandomName()
			appName = RandomName()

			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-app")
			Expect(err).ToNot(HaveOccurred())

			appPath = tmpdir
			CreateBuildPack(BuildpackName, appName, "1.0", 0)
			CreateDeployment(appPath, appName, 0)

		})
	})


	AfterEach(func() {
		DeleteBuildPack(BuildpackName)
		Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		err := os.RemoveAll(appPath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("when it has special characters in app name", func() {

		It("completes successfully with -n option", func() {

			appName = "app!@#$%^&*-name"
			hostName := "special-host-name"
			//specify another buildpack. It will supersede the buildpack in manifest.yml
			push := Cf("push", appName, "-p", appPath, "-n", hostName).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("urls: " + hostName))
			Expect(app).To(Say("#0	 running"))

		})

		It("completes successfully without -n option", func() {

			appName = "app!@#$%^&*-name"
			//specify another buildpack. It will supersede the buildpack in manifest.yml
			push := Cf("push", appName, "-p", appPath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("urls: app-name"))
			Expect(app).To(Say("#0	 running"))

		})


	})


})
