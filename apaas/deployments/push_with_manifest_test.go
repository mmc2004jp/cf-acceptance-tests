package deployments

import (
	"fmt"
	"io/ioutil"
	"strings"
	"os"
	"path"
	"path/filepath"
//	"math/rand"
	"../helpers/assets"
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

		appPath		 string
		manifestFilePath string
		domainName	 string

	)

	createDeploymentUnderSubDir := func(appPath string) {

		abcPath := path.Join(appPath, "abc")
		err := os.Mkdir(abcPath, 0775)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(abcPath, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(path.Join(abcPath, "some-file"))
		Expect(err).ToNot(HaveOccurred())

	}

	createDomain := func(domainName string) {
		AsUser(context.AdminUserContext(), func() {
			Expect(Cf("target", "-o", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete-domain", "-f", domainName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("create-domain", context.RegularUserContext().Org, domainName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	}

	deleteDomain := func(domainName string) {
		AsUser(context.AdminUserContext(), func() {
			Expect(Cf("target", "-o", context.RegularUserContext().Org).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete-domain", "-f", domainName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
	}


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


	Context("when it specifies manifest.yml in different path", func() {

		It("completes successfully without specifying path in manifest", func() {
			randVersion := "1.0"

			CreateDeployment(appPath, appName, 0)

			//create manifest in different path. no path specified
			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-manifest")
			Expect(err).ToNot(HaveOccurred())

			content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)

			manifestFilePath = CreateManifest(tmpdir, "hello_manifest.yml", content)

			//specify manifest file in different path with appPaht
			push := Cf("push", "-p", appPath, "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})

/*
<appRoot>/
	abc/   ...this contains application files
	manifest.yml

manifest contains:
	path: ./abc
*/

		It("completes successfully by specifying relative path to app path if manifest exists in the app path", func() {
			randVersion := "1.0"

			createDeploymentUnderSubDir(appPath)

			//manifest should be in the app path when app path is specified in relative path.
			//otherwise, it will not find the app files
			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: ./abc
`, appName)
			manifestFilePath := CreateManifest(appPath, "hello_manifest.yml", content)

			//specify manifest file in different path with appPaht
			push := Cf("push", appName, "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})

/*
<appRoot>/
	abc/   ...this contains application files
<anotherDir>
	manifest.yml

manifest contains:
	path: ./abc
*/

		It("fails by specifying relative path if manefist exists in different path to app path", func() {
			createDeploymentUnderSubDir(appPath)

			//manifest in different path to app path
			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-manifest")
			Expect(err).ToNot(HaveOccurred())


			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: ./abc
`, appName)
			manifestFilePath := CreateManifest(tmpdir, "hello_manifest.yml", content)


			//specify manifest file in different path with appPath
			push := Cf("push", appName, "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).ToNot(Exit(0))
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("Error uploading application"))
			Expect(push).To(Say("no such file or directory"))
		})


		It("completes successfully by specifying full path even if manefist exists in different path to app path", func() {
			randVersion := "1.0"

			CreateDeployment(appPath, appName, 0)

			//create manifest in different path. no path specified
			tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-manifest")
			Expect(err).ToNot(HaveOccurred())

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: %s
`, appName, appPath)

			manifestFilePath = CreateManifest(tmpdir, "hello_manifest.yml", content)

			//specify manifest file in different path with appPaht
			push := Cf("push", "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: " + randVersion))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})



		AfterEach(func() {
			DeleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())
			err = os.RemoveAll(path.Dir(manifestFilePath))
			Expect(err).ToNot(HaveOccurred())
		})
	})



	Context("when it specifies manifest.yml in app path", func() {

		It("completes successfully with the right setting in manifest", func() {

			CreateDeployment(appPath, appName, 0)

			content := fmt.Sprintf(`
---
applications:
- name: %s
  memory: 512M
`, appName)

			manifestFilePath = CreateManifest(appPath, "hello_manifest.yml", content)

			//specify manifest file in different path with appPaht
			push := Cf("push", "-p", appPath, "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
			Expect(push).To(Say("VERSION: 1.0"))

			Eventually(func() string {
				 return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("urls: " + appName))
			Expect(app).To(Say("of 512M"))

		})


		It("completes successfully with the right services, hosts, env setting in manifest", func() {

			javaAppPath, _ := filepath.Abs(assets.NewAssets().Java)

			content := fmt.Sprintf(`
---
applications:
- name: %s
  memory: 512M
  buildpack: java_buildpack
  domain: example.com
  instances: 4
  hosts:
  - app_host1
  - app_host2
  path: %s
  timeout: 180
  env:
    ENV1: true
    ENV2: 100
  services:
  - service1
  - service2
`, appName, javaAppPath)

			manifestFilePath = CreateManifest(appPath, "hello_manifest.yml", content)

			domainName = "example.com"

			createDomain(domainName)
			Cf("cups", "service1").Wait(DEFAULT_TIMEOUT)
			Cf("cups", "service2").Wait(DEFAULT_TIMEOUT)

			//specify manifest file in different path with appPaht
			push := Cf("push", "-f", manifestFilePath).Wait(LONG_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Creating route app_host1.example.com..."))
			Expect(push).To(Say("Binding app_host1.example.com"))
			Expect(push).To(Say("Creating route app_host2.example.com..."))
			Expect(push).To(Say("Binding app_host2.example.com"))
			Expect(push).To(Say("Done uploading"))
			Expect(push).To(Say("Binding service service1"))
			Expect(push).To(Say("OK"))
			Expect(push).To(Say("Binding service service2"))
			Expect(push).To(Say("OK"))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 4/4"))
			Expect(app).To(Say("usage: 512M x 4 instances"))
			Expect(app).To(Say("app_host1.example.com, app_host2.example.com"))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))
			Expect(app).To(Say("#2"))
			Expect(app).To(Say("#3"))

			appEnv := Cf("env", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(appEnv.Out.Contents()).To(ContainSubstring(fmt.Sprintf("ENV1: true")))
			Expect(appEnv.Out.Contents()).To(ContainSubstring(fmt.Sprintf("ENV2: 100")))

		})

		It("completes successfully with inherited manifest", func() {

			javaAppPath, _ := filepath.Abs(assets.NewAssets().SimpleJavaWar)

			content := fmt.Sprintf(`
---
domain: example.com
memory: 256M
instances: 1

applications:
 - name: springtock
   host: 765shower
   path: %s
 - name: wintertick
   subdomain: 321flurry
   path: %s

`, javaAppPath, javaAppPath)

			manifestFilePath = CreateManifest(appPath, "simple-base-manifest.yml", content)

			content = fmt.Sprintf(`
---
inherit: simple-base-manifest.yml
applications:
 - name: springstorm
   memory: 512M
   instances: 1
   host: 765deluge
   path: %s
 - name: winterblast
   memory: 512M
   instances: 2
   host: 321blizzard
   path: %s

`, javaAppPath, javaAppPath)

			manifestFilePath = CreateManifest(appPath, "simple-prod-manifest.yml", content)


			domainName = "example.com"

			createDomain(domainName)

			//specify manifest file in different path with appPaht
			push := Cf("push", "-f", manifestFilePath).Wait(LONG_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Done uploading"))

			app := Cf("app", "springtock").Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 1/1"))
			Expect(app).To(Say("usage: 256M x 1 instances"))
			Expect(app).To(Say("765shower.example.com"))
			Expect(app).To(Say("#0"))


			app = Cf("app", "wintertick").Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 1/1"))
			Expect(app).To(Say("usage: 256M x 1 instances"))
			Expect(app).To(Say("wintertick.example.com"))
			Expect(app).To(Say("#0"))


			app = Cf("app", "springstorm").Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 1/1"))
			Expect(app).To(Say("usage: 512M x 1 instances"))
			Expect(app).To(Say("765deluge.example.com"))
			Expect(app).To(Say("#0"))

			app = Cf("app", "winterblast").Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 2/2"))
			Expect(app).To(Say("usage: 512M x 2 instances"))
			Expect(app).To(Say("321blizzard.example.com"))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("#1"))


		})

		It("completes successfully with overwriting/adding attributes inherited manifest", func() {

			CreateDeployment(appPath, appName, 0)

			content := fmt.Sprintf(`
---
applications:
 - name: app
   memory: 256M
   disk_quota: 256M
   host: test123
   path: %s

`, appPath)

			manifestFilePath = CreateManifest(appPath, "base.yml", content)

			content = fmt.Sprintf(`
---
inherit: base.yml
applications:
 - name: app
   memory: 512M
   disk_quota: 1G
   host: prod1
   path: %s
   instances: 2

`, appPath)

			manifestFilePath = CreateManifest(appPath, "sub.yml", content)


			//specify manifest file in different path with appPaht
			push := Cf("push", "-f", manifestFilePath).Wait(LONG_TIMEOUT)
			Expect(push).To(Exit(0))
			Expect(push).To(Say("Creating route prod1." + helpers.LoadConfig().AppsDomain))
			Expect(push).To(Say("Done uploading"))

			app := Cf("app", "app").Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 2/2"))
			Expect(app).To(Say("usage: 512M x 2 instances"))
			Expect(app).To(Say("prod1." + helpers.LoadConfig().AppsDomain))
			Expect(app).To(Say("#0"))
			Expect(app).To(Say("of 512M"))
			Expect(app).To(Say("of 1G"))


		})

		It("completes successfully with all optional attributes", func() {

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
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			app := Cf("app", appName).Wait(DEFAULT_TIMEOUT)
			Expect(app).To(Exit(0))
			Expect(app).To(Say("instances: 1/1"))
			Expect(app).To(Say("usage: 1G x 1 instances"))
			Expect(app).To(Say(appName + "." +  helpers.LoadConfig().AppsDomain))
			Expect(app).To(Say("#0"))
			Expect(app.Out.Contents()).To(MatchRegexp("of 1G .* of 1G"))

			appEnv := Cf("env", appName).Wait(DEFAULT_TIMEOUT)
			Expect(appEnv).To(Exit(0))
			Expect(appEnv).To(Say("No user-defined env variables have been set"))
			Expect(appEnv.Out.Contents()).NotTo(ContainSubstring(fmt.Sprintf("VCAP_SERVICES:{}")))

		})

		It("fails without mandotary attributes", func() {

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
			Expect(Cf("delete", "springstorm", "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete", "winterblast", "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete", "springtock", "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete", "wintertick", "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete", "app", "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			deleteDomain(domainName)

			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())


			if strings.HasPrefix(manifestFilePath, "/tmp/") {
				err = os.RemoveAll(path.Dir(manifestFilePath))
			} else {
				err = os.Remove(manifestFilePath)
			}
			Expect(err).ToNot(HaveOccurred())

		})
	})

})
