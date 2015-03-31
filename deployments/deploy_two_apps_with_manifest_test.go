package deployments

import (
	"fmt"
	"path/filepath"
	"io/ioutil"
	"os"
	"path"
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
		appName1 string
		appName2 string
	)

	createDeployment := func(appPath string) {
		appDir1, err := ioutil.TempDir(appPath, "app1")
		Expect(err).ToNot(HaveOccurred())
		appName1 = filepath.Base(appDir1)

		appDir2, err := ioutil.TempDir(appPath, "app2")
		Expect(err).ToNot(HaveOccurred())
		appName2 = filepath.Base(appDir2)

		_, err = os.Create(path.Join(appDir1, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(path.Join(appDir1, "some-file"))
		Expect(err).ToNot(HaveOccurred())


		_, err = os.Create(path.Join(appDir2, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(path.Join(appDir2, "some-file"))
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		AsUser(context.AdminUserContext(), func() {
			BuildpackName = RandomName()
			appName = RandomName()

			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-app")
			Expect(err).ToNot(HaveOccurred())

			appPath = tmpdir
			//create deployment with multiple apps in one  manifest
			createDeployment(appPath)

		})
	})


	Context("when it specifies manifest.yml", func() {

		It("deploys multiple apps within one manifest file", func() {


			CreateBuildPack(BuildpackName, appName, "1.0", 0)

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: ./%s
- name: %s
  path: ./%s
`, appName1, appName1, appName2, appName2)

			manifestFilePath := CreateManifest(appPath, "manifest.yml", content)

			push := Cf("push", "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName1)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName2)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

		})


		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName1, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete", appName2, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
