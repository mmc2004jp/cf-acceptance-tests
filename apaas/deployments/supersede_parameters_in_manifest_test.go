package deployments

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
//	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("Deploy Apps", func() {
	var (
		appName       string
		BuildpackName string

		appPath string
//		manifestFilePath string

//		buildpackPath	string
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

		})
	})


	Context("when it specifies manifest.yml", func() {

		It("supersedes the buildpack by parameter -b in command line", func() {

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: .
  buildpack: %s
`, appName, BuildpackName)
			CreateDeployment(appPath, appName, 0)
			manifestFilePath := CreateManifest(appPath, "manifest.yml", content)


			var randVersion, anotherBuildpack string
			randVersion = fmt.Sprintf( "%2.2f", rand.Float64() * 5)
			anotherBuildpack = RandomName()
			CreateBuildPack(anotherBuildpack, appName, randVersion, 0)

			//specify another buildpack. It will supersede the buildpack in manifest.yml
			push := Cf("push", "-f", manifestFilePath, "-b", anotherBuildpack).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

			DeleteBuildPack(anotherBuildpack)
		})

		It("supersedes instances by parameter -i in command line", func() {
			randVersion := "1.0"
			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: .
  buildpack: %s
  instances: 2
`, appName, BuildpackName)
			CreateDeployment(appPath, appName, 0)
			manifestFilePath := CreateManifest(appPath, "manifest.yml", content)


			//specify instance number. It will supersede instances in manifest.yml
			push := Cf("push", "-f", manifestFilePath, "-i", "3").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
			Expect(app).To(Say("#2"))

		})


		It("supersedes memory by parameter -m in command line", func() {
			randVersion := "1.0"

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: .
  buildpack: %s
  memory: 1G
`, appName, BuildpackName)
			CreateDeployment(appPath, appName, 0)
			manifestFilePath := CreateManifest(appPath, "manifest.yml", content)

			//specify instance number. It will supersede instances in manifest.yml
			push := Cf("push", "-f", manifestFilePath, "-m", "512M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("usage: 512M x 1 instances"))

		})

		It("supersedes hostname by parameter -n in command line", func() {
			randVersion := "1.0"

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: .
  buildpack: %s
  host: hello1
`, appName, BuildpackName)
			CreateDeployment(appPath, appName, 0)
			manifestFilePath := CreateManifest(appPath, "manifest.yml", content)


			//specify instance number. It will supersede instances in manifest.yml
			push := Cf("push", appName, "-f", manifestFilePath, "-n", "hello2").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Eventually(func() string {
				 return helpers.CurlAppRoot("hello2")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})

		It("supersedes hosts by parameter -n in command line", func() {
			randVersion := "1.0"

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: .
  buildpack: %s
  hosts:
  - hello1
  - hello2
`, appName, BuildpackName)
			CreateDeployment(appPath, appName, 0)
			manifestFilePath := CreateManifest(appPath, "manifest.yml", content)


			//specify instance number. It will supersede instances in manifest.yml
			push := Cf("push", "-f", manifestFilePath, "-n", "hello3").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Eventually(func() string {
				 return helpers.CurlAppRoot("hello3")
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})

		It("supersedes path by parameter -p in command line", func() {
			randVersion := "1.0"

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: %s
  buildpack: %s
`, appName, "do-not-exist-path", BuildpackName)
			CreateDeployment(appPath, appName, 0)
			manifestFilePath := CreateManifest(appPath, "manifest.yml", content)


			//specify instance number. It will supersede instances in manifest.yml
			push := Cf("push", "-p", appPath, "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})

		It("supersedes timeout by parameter -t in command line", func() {
			randVersion := "1.0"

			content := fmt.Sprintf(`
---
applications:
- name: %s
  timeout: 180
  buildpack: %s
`, appName, BuildpackName)
			CreateDeployment(appPath, appName, 0)
			CreateManifest(appPath, "manifest.yml", content)


			command := "sleep 30; while true; do { echo -e 'HTTP/1.1 200 OK\r\n';echo \"hi from a simple admin buildpack\";} | nc -l $PORT; done"

			//specify timeout number. It will supersede timeout in manifest.yml
			push := Cf("push", appName, "-t", "30", "-p", appPath, "-c", command ).Wait(CF_PUSH_TIMEOUT)
			Expect(push).NotTo(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))
			Eventually(func() *Session {
				appLogsSession := Cf("logs", "--recent",appName)
				Expect(appLogsSession.Wait(DEFAULT_TIMEOUT)).To(Exit(0))
					return appLogsSession
			}, 30).Should(Say("failed to accept connections within health check timeout"))

		})

		It("supersedes disk quota by parameter -k in command line", func() {
			randVersion := "1.0"

			content := fmt.Sprintf(`
---
applications:
- name: %s
  disk_quota: 1G
  buildpack: %s
`, appName, BuildpackName)
			CreateDeployment(appPath, appName, 0)
			CreateManifest(appPath, "manifest.yml", content)


			//specify timeout number. It will supersede timeout in manifest.yml
			push := Cf("push", appName, "-p", appPath, "-k", "900M").Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("of 900M"))

		})



		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
