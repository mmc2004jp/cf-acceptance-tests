package buildpacks

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

			CreateBuildPack(BuildpackName, appName, "1.0", 0, false)
			CreateDeployment(appPath, appName, 0)
		})
	})


	Context("when the buildpack is updated", func() {

		It("is used the old version for the running app", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, false)

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


		})


		It("is used the old version for the app even after stopping and starting again", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, false)

			Expect(Cf("stop", appName).Wait(LONG_TIMEOUT)).To(Exit(0))
			Expect(Cf("start", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})

		It("is used the old version for the app even after restarting", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, false)

			Expect(Cf("restart", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})

		It("is used the old version for the app even after pushing more instances and restarting one of them", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M", "-i", "2").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, false)

			Expect(Cf("restart-app-instance", appName, "0").Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})

		It("is used the new version for the app after pushing with no-start and then starting", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, false)

			Expect(Cf("push", appName, "-p", appPath, "-m", "128M", "--no-start").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
			Expect(Cf("start", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 2.0"))

		})


		It("is used the new version for the app after pushing again", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, false)

			Expect(Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 2.0"))

		})

		It("is used the new version for the app after restaging", func() {
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			UpdateBuildPack(BuildpackName, appName, "2.0", 0, false)

			Expect(Cf("restage", appName).Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 2.0"))

		})



		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
