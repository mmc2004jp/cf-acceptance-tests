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

var _ = Describe("Set Environment", func() {
	var (
		appName       string
		BuildpackName string

		appPath string
//		manifestFilePath string

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

		})
	})


	AfterEach(func() {
		DeleteBuildPack(BuildpackName)
		Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		err := os.RemoveAll(appPath)
		Expect(err).ToNot(HaveOccurred())

	})

	//chapter 6 - 1.1
	It("prints user-defined variable value successfully", func() {
		randVersion := "1.0"
		CreateBuildPack(BuildpackName, appName, randVersion, 0)

		content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
		CreateDeployment(appPath, appName, 0)
		CreateManifest(appPath, appName, content)

		Expect(Cf("push", appName, "-p", appPath, "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(Cf("set-env", appName, "MY_ENV", "this is a user-provided variable").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(Cf("restart", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		var curlResponse string
		Eventually(func() string {
			curlResponse = helpers.CurlAppRoot(appName)
			return curlResponse
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		Expect(curlResponse).To(MatchRegexp("MY_ENV:this is a user-provided variable"))
	})

	//chapter 6 - 1.2, 1.3
	It("prints expected VCAP_SERVICE successfully when binding/unbinding services", func() {
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
		Expect(Cf("cups", serviceName, "-p", "'{\"username\":\"admin\",\"password\":\"pa55woRD\"}'").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		//bind service, VCAP_SERVICE will be applied
		Expect(Cf("bind-service", appName, serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Expect(Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		var curlResponse string
		Eventually(func() string {
			curlResponse = helpers.CurlAppRoot(appName)
			return curlResponse
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		Expect(curlResponse).To(MatchRegexp("VCAP_SERVICES:{.+\\\\password\\\\:\\\\pa55woRD\\\\.+}"))
		Expect(curlResponse).To(MatchRegexp("VCAP_SERVICES:{.+\\\\username\\\\:\\\\admin\\\\.+}"))
		Expect(curlResponse).To(MatchRegexp("MY_ENV:[^$]"))


		//unbind service, VCAP_SERVICES will be cleared
		Expect(Cf("unbind-service", appName, serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Expect(Cf("restart", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			curlResponse = helpers.CurlAppRoot(appName)
			return curlResponse
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


		Expect(curlResponse).To(MatchRegexp("VCAP_SERVICES:{}"))
		Expect(curlResponse).To(MatchRegexp("MY_ENV:[^$]"))

	})
})
