package apps

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

var _ = Describe("Admin Buildpacks", func() {
	var (
		appName       string
		BuildpackName string

		appPath string

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
					Body: `#!/usr/bin/env bash

ms1=$(date +%s)
cat <<EOF
---
config_vars:
  PATH: bin:/usr/local/bin:/usr/bin:/bin
  FROM_BUILD_PACK: "yes"
default_process_types:
  web: while true;do { echo -e 'HTTP/1.1 200 OK\r\n'; echo "hi from a simple admin buildpack"; sleep 1; NOW=\$((\$(date +"%s") - $ms1)); echo "\$NOW"; if [ \$NOW -ge 30 ]; then echo "killing myself"; kill \$$; fi; } | nc -l \$PORT; done

EOF
`,  
				}, 
			})
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
		})
        }

        updateBuildPack := func(buildPackName string, version string) {

                AsUser(context.AdminUserContext(), func() {
                        var err error
                        var tmpdir string

                        tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
                        Expect(err).ToNot(HaveOccurred())

                        buildpackPath = tmpdir
                        buildpackArchivePath = path.Join(buildpackPath, "buildpack_" + version +".zip")

                        createZipArchive(buildpackArchivePath, version)

                        updateBuildpack := Cf("update-buildpack", buildPackName, "-p", buildpackArchivePath).Wait(DEFAULT_TIMEOUT)
                        Expect(updateBuildpack).Should(Exit(0))
                        Expect(updateBuildpack).Should(Say("Done uploading"))
                        Expect(updateBuildpack).Should(Say("OK"))
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

			tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
			Expect(err).ToNot(HaveOccurred())

			buildpackPath = tmpdir
			buildpackArchivePath = path.Join(buildpackPath, "buildpack.zip")

			createZipArchive(buildpackArchivePath, "1.0")

			_, err = os.Create(path.Join(appPath, matchingFilename(appName)))
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Create(path.Join(appPath, "some-file"))
			Expect(err).ToNot(HaveOccurred())

		})
	})


	Context("when the app is crashed", func() {

		It("is used the new version after updating buildpack and then pushing again", func() {
			createBuildPack(BuildpackName, "1.0")	
			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: 1.0")) 

			//the app will response with the message
                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))


			//the app will be killed in 30 seconds
                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, LONG_TIMEOUT).Should(ContainSubstring("404 Not Found"))

			updateBuildPack(BuildpackName, "2.0")
			//will choose the new version buildpack - 2.0
			anotherPush := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
                        Expect(anotherPush).To(Exit(0))
			Expect(anotherPush).To(Say("Staging with Simple Buildpack"))
                 	Expect(anotherPush).To(Say("VERSION: 2.0")) 


		        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                        }, LONG_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))

			
		})
	
		It("is used the new version after deleting buildpack and then pushing again", func() {
			createBuildPack(BuildpackName, "1.0")	

                        var randVersion, anotherBuildPack string
                        randVersion = fmt.Sprintf( "%2.2f", rand.Float64() * 5)
                        anotherBuildPack = RandomName()

                        //the new buildpack always takes 0 position
                        createBuildPack(anotherBuildPack, randVersion)

			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: " + randVersion)) 

			//the app will response with the message
                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))


			//the app will be killed in 30 seconds
                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, LONG_TIMEOUT).Should(ContainSubstring("404 Not Found"))

			deleteBuildPack(anotherBuildPack)
			//will choose the lowest positioned buildpack - 1.0
			anotherPush := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
	                Expect(anotherPush).To(Exit(0))
			Expect(anotherPush).To(Say("Staging with Simple Buildpack"))
                 	Expect(anotherPush).To(Say("VERSION: 1.0")) 

		        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                        }, LONG_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))

			
		})
		
	
				
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
	        })	
	})
})
