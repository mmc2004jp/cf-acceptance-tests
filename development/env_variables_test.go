package development

import (
	"fmt"
	"io/ioutil"
	"os"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
//	. "github.com/onsi/gomega/gbytes"
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

	//chapter 4 - 1
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
			Expect(Cf("set-env", appName, "MY_ENV", "this is a user-provided variable").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
			Expect(Cf("restart", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	


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
			Expect(curlResponse).To(MatchRegexp("MY_ENV:this is a user-provided variable"))


			if CF_VERSION >= 196 {
				Expect(curlResponse).To(MatchRegexp("CF_INSTANCE_ADDR:[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.:[0-9]+"))
				Expect(curlResponse).To(MatchRegexp("CF_INSTANCE_INDEX:0"))
				Expect(curlResponse).To(MatchRegexp("CF_INSTANCE_IP:[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}[0-9].[0-9]{1,3}"))
				Expect(curlResponse).To(MatchRegexp("CF_INSTANCE_PORT:[0-9]+"))
				Expect(curlResponse).To(MatchRegexp("CF_INSTANCE_PORTS:[{external:[0-9]+,internal:[0-9]+"))
			}
		})

		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
