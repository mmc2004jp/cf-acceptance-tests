package buildpacks

import (
	"fmt"
	"io/ioutil"
	"os"
//	"path"
	"math/rand"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = Describe("Admin Buildpacks", func() {
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
			CreateDeployment(appPath, appName, 0)
			CreateBuildPack(BuildpackName, appName, "1.0", 0, true)
		})
	})


	Context("when the app is crashed", func() {

		It("is used the new version after updating buildpack and then pushing again", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			//the app will response with the message
			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT, 5).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


			//the app will be killed in 30 seconds
			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, LONG_TIMEOUT).Should(ContainSubstring("404 Not Found"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, true)
			//will choose the new version buildpack - 2.0
			anotherPush := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(anotherPush).To(Exit(0))
			Expect(anotherPush).To(Say("Staging with Simple Buildpack"))
			Expect(anotherPush).To(Say("VERSION: 2.0"))


			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, LONG_TIMEOUT, 5).Should(ContainSubstring("hi from a simple admin buildpack 2.0"))


		})

		It("is used the new version after deleting buildpack and then pushing again", func() {

			var randVersion, anotherBuildPack string
			randVersion = fmt.Sprintf( "%2.2f", rand.Float64() * 5)
			anotherBuildPack = RandomName()

			//the new buildpack always takes 0 position
			CreateBuildPack(anotherBuildPack, appName, randVersion, 0, true)

			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			//the app will response with the message
			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT, 5).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))


			//the app will be killed in 30 seconds
			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, LONG_TIMEOUT).Should(ContainSubstring("404 Not Found"))

			DeleteBuildPack(anotherBuildPack)
			//will choose the lowest positioned buildpack - 1.0
			anotherPush := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(anotherPush).To(Exit(0))
			Expect(anotherPush).To(Say("Staging with Simple Buildpack"))
			Expect(anotherPush).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, LONG_TIMEOUT, 5).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


		})



		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
