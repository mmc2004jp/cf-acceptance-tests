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
	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("Deploy Apps", func() {
	var (
		appName       string
		BuildpackName string

		appPath          string
		manifestFilePath string
		domainName       string

		buildpackPath        string
		buildpackArchivePath string
	)

	matchingFilename := func(appName string) string {
		return fmt.Sprintf("simple-buildpack-please-match-%s", appName)
	}

	createZipArchive := func(builpackArchivePath string, version string) { 
			archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
				{
					Name: "bin/compile",
					Body: fmt.Sprintf(`#!/usr/bin/env bash

sleep 1 # give loggregator time to start streaming the logs

echo "Staging with Simple Buildpack"
echo  "VERSION: %s" 

sleep 10
`, version),
				},
				{
					Name: "bin/detect",
					Body: fmt.Sprintf(`#!/bin/bash

if [ -f "${1}/%s" ]; then
  echo Simple
else
  echo no
  exit 1
fi
`, matchingFilename(appName)),
				},
				{
					Name: "bin/release",
					Body: fmt.Sprintf( 
`#!/usr/bin/env bash

buildpackVersion="%s"
cat <<EOF
---
config_vars:
  PATH: bin:/usr/local/bin:/usr/bin:/bin
  FROM_BUILD_PACK: "yes"
default_process_types:
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n';echo "hi from a simple admin buildpack $buildpackVersion";} | nc -l \$PORT; done
EOF
`, version), 
				}, 
			})
}

	createDeployment := func(appPath string) {

		_, err := os.Create(path.Join(appPath, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(path.Join(appPath, "some-file"))
		Expect(err).ToNot(HaveOccurred())
	
	}


	createDeploymentUnderSubDir := func(appPath string) {

                abcPath := path.Join(appPath, "abc")
                err := os.Mkdir(abcPath, 0775)
                Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(abcPath, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(path.Join(abcPath, "some-file"))
		Expect(err).ToNot(HaveOccurred())

	}


	createManifest := func(manifestPath string, manifestFileName string, content string) {

		manifestFile, err := os.Create(path.Join(manifestPath, manifestFileName))
		Expect(err).ToNot(HaveOccurred())
		manifestFilePath = manifestFile.Name()

		_, err = manifestFile.WriteString(content)
		Expect(err).ToNot(HaveOccurred())

	}

	createBuildPack := func(buildPackName string, version string) { 
		
		AsUser(context.AdminUserContext(), func() {
                        var err error
                        var tmpdir string

			tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack_" + version + ".zip")

			createZipArchive(buildpackArchivePath, version)

			createBuildpack := Cf("create-buildpack", buildPackName, buildpackArchivePath, "0").Wait(DEFAULT_TIMEOUT)
			Expect(createBuildpack).Should(Exit(0))
			Expect(createBuildpack).Should(Say("Creating"))
			Expect(createBuildpack).Should(Say("OK"))
			Expect(createBuildpack).Should(Say("Uploading"))
			Expect(createBuildpack).Should(Say("OK"))

			//clean the temporary directory of the buildpack 
			err = os.RemoveAll(buildpackPath)
			Expect(err).ToNot(HaveOccurred())			
		})
        }
	
	deleteBuildPack := func(buildpackName string) { 
		
		AsUser(context.AdminUserContext(), func() {
			Expect(Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})
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
			createBuildPack(BuildpackName, "1.0")	

		})
	})


	Context("when it specifies manifest.yml in different path", func() {
	
		It("completes successfully without specifying path in manifest", func() {
			randVersion := "1.0"

			createDeployment(appPath)

	                //create manifest in different path. no path specified
        	       	tmpdir, err := ioutil.TempDir(os.TempDir(), "matching-manifest")
                	Expect(err).ToNot(HaveOccurred())

			content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)

			createManifest(tmpdir, "manifest.yml", content)

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
			createManifest(appPath, "manifest.yml", content)

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
			createManifest(tmpdir, "manifest.yml", content)


			//specify manifest file in different path with appPath
			push := Cf("push", appName, "-f", manifestFilePath).Wait(CF_PUSH_TIMEOUT)
                        Expect(push).ToNot(Exit(0))
			Expect(push).To(Say("FAILED"))
                  	Expect(push).To(Say("Error uploading application")) 
                 	Expect(push).To(Say("no such file or directory")) 
		})
								
				
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())			
			err = os.RemoveAll(path.Dir(manifestFilePath))
			Expect(err).ToNot(HaveOccurred())			
	        })	
	})



	Context("when it specifies manifest.yml in app path", func() {
	
		It("completes successfully with the right setting in manifest", func() {

			createDeployment(appPath)

			content := fmt.Sprintf(`
---
applications:
- name: %s
  memory: 512M
`, appName)

			createManifest(appPath, "manifest.yml", content)

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

			createManifest(appPath, "manifest.yml", content)

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

			createManifest(appPath, "simple-base-manifest.yml", content)

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

			createManifest(appPath, "simple-prod-manifest.yml", content)


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

			createDeployment(appPath)

			content := fmt.Sprintf(`
---
applications:
 - name: app
   memory: 256M
   disk_quota: 256M
   host: test123
   path: %s

`, appPath)

			createManifest(appPath, "base.yml", content)

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

			createManifest(appPath, "sub.yml", content)


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
	
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
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
