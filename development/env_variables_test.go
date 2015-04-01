package development

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
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
)

var _ = Describe("Deploy Apps", func() {
	var (
		appName       string
		BuildpackName string

		appPath string
//		manifestFilePath string

//		buildpackPath        string
//		buildpackArchivePath string
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


	Context("when it prints environment variables", func() {

		It("completes successfully", func() {
			randVersion := "1.0"
			CreateBuildPack(BuildpackName, appName, randVersion, 0)

			content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
			CreateDeployment(appPath, appName, 0)
			CreateManifest(appPath, appName, content)


			Expect(Cf("push", appName, "-p", appPath, "--no-start", "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			serviceName := RandomName()
			Expect(Cf("cups", serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			//bind service, VCAP_SERVICE will be applied
			Expect(Cf("bind-service", appName, serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Expect(Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))


			var curlResponse string
			Eventually(func() string {
				curlResponse = helpers.CurlAppRoot(appName)
				return curlResponse
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))

			Expect(curlResponse).To(ContainSubstring("HOME:/home/vcap/app"))
			Expect(curlResponse).To(ContainSubstring("MEMORY_LIMIT:512m"))
			Expect(curlResponse).To(MatchRegexp("PORT:[0-9]+"))
			Expect(curlResponse).To(ContainSubstring("PWD:/home/vcap/app"))
			Expect(curlResponse).To(ContainSubstring("TMPDIR:/home/vcap/tmp"))
			Expect(curlResponse).To(ContainSubstring("USER:vcap"))
			Expect(curlResponse).To(ContainSubstring("VCAP_APP_HOST:0.0.0.0"))
			Expect(curlResponse).To(MatchRegexp("VCAP_APPLICATION:{.+}"))
			Expect(curlResponse).To(MatchRegexp("VCAP_APP_PORT:[0-9]+"))
			Expect(curlResponse).To(MatchRegexp("VCAP_SERVICES:{.+}"))

		})


		It("completes successfully with all optional attributes", func() {
			randVersion := "1.0"
			CreateBuildPack(BuildpackName, appName, randVersion, 0)

			content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
			CreateDeployment(appPath, appName, 0)
			CreateManifest(appPath, appName, content)

			Expect(Cf("push", appName, "-p", appPath).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

			var curlResponse string
			Eventually(func() string {
				curlResponse = helpers.CurlAppRoot(appName)
				return curlResponse
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 1/1"))
			Expect(app).To(Say("usage: 1G x 1 instances"))
			Expect(app).To(Say(appName + "." +  helpers.LoadConfig().AppsDomain))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("of 1G"))
			Expect(app).To(Say("of 1G"))

			appEnv := Cf("env", appName).Wait(DEFAULT_TIMEOUT)
			Expect(appEnv).To(Exit(0))
			Expect(appEnv).To(Say("No user-defined env variables have been set"))
			Expect(appEnv.Out.Contents()).NotTo(ContainSubstring(fmt.Sprintf("VCAP_SERVICES:{}")))

		})

		It("fails without mandotary attributes", func() {
			randVersion := "1.0"
			CreateBuildPack(BuildpackName, appName, randVersion, 0)

			content := fmt.Sprintf(`
---
applications:
- memory: 512M
`)
			CreateDeployment(appPath, appName, 0)
			manifestFilePath := CreateManifest(appPath, appName, content)

			push := Cf("push", "-p", appPath, "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(1))
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("Error: App name is a required field"))

		})
		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
