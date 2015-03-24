package deployments

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"math/rand"
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

		appPath string
		manifestFilePath string

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

	createDeployment := func(appPath string, manifestContent string) {

		_, err := os.Create(path.Join(appPath, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(path.Join(appPath, "some-file"))
		Expect(err).ToNot(HaveOccurred())

		manifestFile, err := os.Create(path.Join(appPath, "manifest.yml"))
		Expect(err).ToNot(HaveOccurred())
		manifestFilePath = manifestFile.Name()

		_, err = manifestFile.WriteString(manifestContent)

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


	Context("when it specifies manifest.yml", func() {
	
		It("supersedes the buildpack by parameter -b in command line", func() {

			content := fmt.Sprintf(`
---
applications:
- name: %s
  path: .
  buildpack: %s
`, appName, BuildpackName)
			createDeployment(appPath, content)


			var randVersion, anotherBuildpack string
                        randVersion = fmt.Sprintf( "%2.2f", rand.Float64() * 5)
			anotherBuildpack = RandomName()
			createBuildPack(anotherBuildpack, randVersion)	

			//specify another buildpack. It will supersede the buildpack in manifest.yml
			push := Cf("push", "-f", manifestFilePath, "-b", anotherBuildpack).Wait(CF_PUSH_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: " + randVersion)) 

                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

			deleteBuildPack(anotherBuildpack)	
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
			createDeployment(appPath, content)


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
			createDeployment(appPath, content)

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
			createDeployment(appPath, content)

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
			createDeployment(appPath, content)

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
			createDeployment(appPath, content)

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
                        createDeployment(appPath, content)

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

				
			
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())			
	        })	
	})

})
