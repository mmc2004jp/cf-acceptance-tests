package deployments

import (
//	"fmt"
	"io/ioutil"
	"os"
//	"path"
//	"math/rand"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = Describe("Deploy Apps", func() {
	var (
		appName       string
		BuildpackName string

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

		})
	})


	Context("when its size is huge", func() {

		It("can complete successfully under 1G ", func() {
			//create app with 990M
			CreateDeployment(appPath, appName, 990*1024*1024)
			randVersion := "1.0"

			push := Cf("push", appName, "-p", appPath).Wait(LONG_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})

		It("fails to push app over 1G size", func() {
			//create app with 1.1G
			CreateDeployment(appPath, appName, 1100*1024*1024)

			push := Cf("push", appName, "-p", appPath).Wait(LONG_TIMEOUT)
			Expect(push).ToNot(Exit(0))
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("The app package is invalid: Package may not be larger than 1073741824 bytes"))

		})

		It("completes successfully with -k parameter smaller than the real app size under 1G", func() {
			//create app with 490M
			CreateDeployment(appPath, appName, 490*1024*1024)

			randVersion := "1.0"

			push := Cf("push", appName, "-p", appPath, "-k", "500M").Wait(LONG_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})

		It("fails with -k parameter smaller than the real app size under 1G", func() {
			//create app with 510M
			CreateDeployment(appPath, appName, 510*1024*1024)

			randVersion := "1.0"
			push := Cf("push", appName, "-p", appPath, "-k", "500M").Wait(LONG_TIMEOUT)
			Expect(push).ToNot(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("Start unsuccessful"))


		})


		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
