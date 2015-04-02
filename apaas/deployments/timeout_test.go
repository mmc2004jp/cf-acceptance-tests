package deployments

import (
	"fmt"
	"io/ioutil"
	"os"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
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
			//createBuildPack(BuildpackName, "1.0")

		})
	})


	Context("when it timeouts", func() {

		It("should timeout after 5 minutes in starting phase", func() {
			randVersion := "1.0"

			CreateBuildPack(BuildpackName, appName, randVersion, 10)


			content := fmt.Sprintf(`
---
applications:
- name: %s
  buildpack: %s
`, appName, BuildpackName)

			CreateDeployment(appPath, appName, 0)
			CreateManifest(appPath, "manifest.yml", content)

			//sleep 310 seconds
			command := "sleep 310; while true; do { echo -e 'HTTP/1.1 200 OK\r\n';echo \"hi from a simple admin buildpack\";} | nc -l $PORT; done"

			//specify timeout number. It will supersede timeout in manifest.yml
			push := Cf("push", appName, "-p", appPath, "--no-start", "-c", command ).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))

			start := Cf("start", appName).Wait(LONG_TIMEOUT)
			Expect(start).To(Say("Staging with Simple Buildpack"))
			Expect(start).To(Say("VERSION: " + randVersion))
			Expect(start).To(Say("FAILED"))
			Expect(start).To(Say("Start unsuccessful"))

			Eventually(func() *Session {
				appLogsSession := Cf("logs", "--recent",appName)
				Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
					return appLogsSession
			}, DEFAULT_TIMEOUT).Should(Say("failed to accept connections within health check timeout"))

		})

		It("timeouts staging after 15 minutes by default", func() {
			randVersion := "1.0"
			CreateBuildPack(BuildpackName, appName, randVersion, 15 * 60)

			content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
			CreateDeployment(appPath, appName, 0)
			CreateManifest(appPath, "manifest.yml", content)


			//specify timeout number. It will supersede timeout in manifest.yml
			push := Cf("push", appName, "-p", appPath, "-m", "512M").Wait(LONG_TIMEOUT_20)

			// exit abnormally
			Expect(push).To(Exit(1))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))
			Expect(push).To(Say("Sleeping 900s"))
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("Staging error: cannot get instances since staging failed"))

			Eventually(func() *Session {
				appLogsSession := Cf("logs", "--recent",appName)
				Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
					return appLogsSession
			}, DEFAULT_TIMEOUT).Should(Say("Staging error: failed to stage application"))

		})

		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
