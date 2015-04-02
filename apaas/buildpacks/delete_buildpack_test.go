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

		})
	})


	Context("when the buildpack is deleted", func() {

		It("is used the old version for the running app", func() {
			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			DeleteBuildPack(BuildpackName)

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


		})


		It("is used the old version for the app even after stopping and starting again", func() {
			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			DeleteBuildPack(BuildpackName)

			Expect(Cf("stop", appName).Wait(LONG_TIMEOUT)).To(Exit(0))
			Expect(Cf("start", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})

		It("is used the old version for the app even after restarting", func() {
			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			DeleteBuildPack(BuildpackName)

			Expect(Cf("restart", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})

		It("is used the old version for the app even after pushing more instances and restarting one of them", func() {
			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)
			push := Cf("push", appName, "-p", appPath, "-m", "128M", "-i", "2").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			DeleteBuildPack(BuildpackName)

			Expect(Cf("restart-app-instance", appName, "0").Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})

		It("is used the lowest positioned buildpack version for the app after pushing with no-start and then starting", func() {
			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)

			var randVersion, anotherBuildPack string
			randVersion = fmt.Sprintf( "%2.2f", rand.Float64() * 5)
			anotherBuildPack = RandomName()

			//the new buildpack always takes 0 position
			CreateBuildPack(anotherBuildPack, appName, randVersion, 0, false)

			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

			DeleteBuildPack(anotherBuildPack)

			Expect(Cf("push", appName, "-p", appPath, "-m", "128M", "--no-start").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
			Expect(Cf("start", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})


		It("is used the new version for the app after pushing again", func() {
			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)

			var randVersion, anotherBuildPack string
			randVersion = fmt.Sprintf( "%2.2f", rand.Float64() * 5)
			anotherBuildPack = RandomName()

			//the new buildpack always takes 0 position
			CreateBuildPack(anotherBuildPack, appName, randVersion, 0, false)

			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

			DeleteBuildPack(anotherBuildPack)

			Expect(Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})

		It("is used the new version for the app after restaging", func() {
			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)

			var randVersion, anotherBuildPack string
			randVersion = fmt.Sprintf( "%2.2f", rand.Float64() * 5)
			anotherBuildPack = RandomName()

			//the new buildpack always takes 0 position
			CreateBuildPack(anotherBuildPack, appName, randVersion, 0, false)

			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

			DeleteBuildPack(anotherBuildPack)

			Expect(Cf("restage", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})


		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			err := os.RemoveAll(appPath)
			Expect(err).NotTo(HaveOccurred())

		})
	})

	Context("when the buildpack is not detected", func() {

		It("fails to push app", func() {
			CreateDeployment(appPath, appName, 0)
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).ToNot(Exit(0))
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("An app was not successfully detected by any available buildpack"))

		})


		AfterEach(func() {
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			err := os.RemoveAll(appPath)
			Expect(err).NotTo(HaveOccurred())

		})
	})
})
