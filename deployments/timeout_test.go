package deployments

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
//	"math/rand"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
//        "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
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

	createZipArchive := func(builpackArchivePath string, version string, timeout int64) { 
			archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
				{
					Name: "bin/compile",
					Body: fmt.Sprintf(`#!/usr/bin/env bash

sleep 1 # give loggregator time to start streaming the logs

echo "Staging with Simple Buildpack"
echo "VERSION: %s" 
echo "Sleeping %ds..."
sleep %d 
echo "wake up...."

`, version, timeout, timeout),
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

        createDeployment := func(appPath string, size int64)  {

                _, err := os.Create(path.Join(appPath, matchingFilename(appName)))
                Expect(err).ToNot(HaveOccurred())
                appFile, err := os.Create(path.Join(appPath, "some-file"))
                Expect(err).ToNot(HaveOccurred())
                err = appFile.Truncate(size)
                Expect(err).ToNot(HaveOccurred())

        }

        createManifest := func(manifestPath string, manifestContent string) {
                manifestFile, err := os.Create(path.Join(manifestPath, "manifest.yml"))
                Expect(err).ToNot(HaveOccurred())
                manifestFilePath = manifestFile.Name()

                _, err = manifestFile.WriteString(manifestContent)

                Expect(err).ToNot(HaveOccurred())
        }

        
	createBuildPack := func(buildPackName string, version string, timeout int64) { 
		
		AsUser(context.AdminUserContext(), func() {
                        var err error
                        var tmpdir string

			tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack_" + version + ".zip")

			createZipArchive(buildpackArchivePath, version, timeout)

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
			//createBuildPack(BuildpackName, "1.0")	

		})
	})


	Context("when it timeouts", func() {

               It("should timeout with the expected error message", func() {
                        randVersion := "1.0"
                        createBuildPack(BuildpackName, randVersion, 2 * 60)

                        content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
                        createDeployment(appPath, 0)
			createManifest(appPath, content)


                        // exit abnormally, if waiting for a short time DEFAULT_TIMEOUT (30s)
                        failures := InterceptGomegaFailures(func() {
                                push := Cf("push", appName, "-p", appPath, "-m", "512M").Wait(DEFAULT_TIMEOUT)
 				Expect(push).ShouldNot(Exit())
                                Expect(push).Should(Exit())
                        })

  			Expect(failures[0]).Should(ContainSubstring("Timed out"))
	                Expect(failures[0]).Should(ContainSubstring("Expected process to exit.  It did not."))

                })

               It("should timeout after 5 minutes in starting phase", func() {
                        randVersion := "1.0"

			createBuildPack(BuildpackName, randVersion, 10)


                        content := fmt.Sprintf(`
---
applications:
- name: %s
  buildpack: %s
`, appName, BuildpackName)
                        createDeployment(appPath, 0)
			createManifest(appPath, content)

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
			createBuildPack(BuildpackName, randVersion, 15 * 60)

			content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
			createDeployment(appPath, 0)
			createManifest(appPath, content)


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

                It("timeouts staging by specifying CF_STAGING_TIMEOUT", func() {
                        env := make(map[string]string)
                        env["CF_STAGING_TIMEOUT"] = "3"

                        AsInterceptCommand("", env, func() {
                                randVersion := "1.0"
                                createBuildPack(BuildpackName, randVersion, 15 * 60)

                                content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
                                createDeployment(appPath, 0)
                                createManifest(appPath, content)


                                //specify timeout number. It will supersede timeout in manifest.yml
                                push := Cf("push", appName, "-p", appPath, "-m", "512M").Wait(LONG_TIMEOUT_20)

                                // exit abnormally
                                Expect(push).To(Exit(1))
                                Expect(push).To(Say("Staging with Simple Buildpack"))
                                Expect(push).To(Say("VERSION: " + randVersion))
                                Expect(push).To(Say("Sleeping 900s"))
                                Expect(push).To(Say("FAILED"))
                                Expect(push).To(Say("failed to stage within 3.000000 minutes"))


                        })
                })
		
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())			
	        })	
	})

})
