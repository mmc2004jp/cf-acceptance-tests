package deployments

import (
	"fmt"
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

var _ = Describe("Deploy Apps", func() {
	var (
		appName       string
		BuildpackName string

		appPath string
		manifestFilePath string

//		buildpackPath	     string
//		buildpackArchivePath string
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

	Context("when it specifies no-route in manifest.yml", func() {

		It("will not assign a route to the application", func() {
			//create a manifest file with no-route config
			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: .
  no-route: true
`, appName)
			manifestFilePath = CreateManifest(appPath, appName, content)

			//specify another buildpack. It will supersede the buildpack in manifest.yml
			push := Cf("push", "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() *Session {
				session := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				return session
			}, DEFAULT_TIMEOUT).Should(Say("#0   running"))


			Eventually(func() string {
				session := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				return fmt.Sprintf("%s", session.Out.Contents)
			}, DEFAULT_TIMEOUT, 5).ShouldNot(ContainSubstring(helpers.LoadConfig().AppsDomain))

		})


	})


	Context("when it specifies no-route in command line", func() {

		It("will not assign a route to the application", func() {

			//specify another buildpack. It will supersede the buildpack in manifest.yml
			push := Cf("push", appName, "-p", appPath, "--no-route").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() *Session {
				session := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				return session
			}, DEFAULT_TIMEOUT).Should(Say("#0   running"))


			Eventually(func() string {
				session := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				return fmt.Sprintf("%s", session.Out.Contents)
			}, DEFAULT_TIMEOUT, 5).ShouldNot(ContainSubstring(helpers.LoadConfig().AppsDomain))

		})

	})


	Context("when it unmap route", func() {

		It("will remove the route from the app", func() {
			randVersion := "1.0"

			//specify another buildpack. It will supersede the buildpack in manifest.yml
			push := Cf("push", appName, "-p", appPath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))


			Expect(Cf("unmap-route", appName, helpers.LoadConfig().AppsDomain, "-n", appName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Eventually(func() *Session {
				session := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				return session
			}, DEFAULT_TIMEOUT).Should(Say("#0   running"))


			Eventually(func() string {
				session := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
				Expect(session.Wait(DEFAULT_TIMEOUT)).To(Exit(0))

				return fmt.Sprintf("%s", session.Out.Contents)
			}, DEFAULT_TIMEOUT, 5).ShouldNot(ContainSubstring(helpers.LoadConfig().AppsDomain))

		})

	})


})
