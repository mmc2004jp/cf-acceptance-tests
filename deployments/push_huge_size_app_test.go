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
        "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("Deploy Apps", func() {
	var (
		appName       string
		BuildpackName string

		appPath string
	//	manifestFilePath string

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

	createDeployment := func(appPath string, size int64) {

		appFile, err := os.Create(path.Join(appPath, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		appFile, err = os.Create(path.Join(appPath, "some-file"))
		Expect(err).ToNot(HaveOccurred())
		err = appFile.Truncate(size)
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


	Context("when its size is huge", func() {
	
		It("can complete successfully under 1G ", func() {
			//create app with 100M	
			createDeployment(appPath, 100*1024*1024)			
			randVersion := "1.0"

			push := Cf("push", appName, "-p", appPath).Wait(LONG_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: " + randVersion)) 

                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})
								
		It("fails to push app over 1G size", func() {
			//create app with 1.1G	
			createDeployment(appPath, 1100*1024*1024)			

			push := Cf("push", appName, "-p", appPath).Wait(LONG_TIMEOUT)
                        Expect(push).ToNot(Exit(0))
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("The app package is invalid: Package may not be larger than 1073741824 bytes"))

		})
				
		It("completes successfully with -k parameter smaller than the real app size under 1G", func() {
			//create app with 490M
			createDeployment(appPath, 490*1024*1024)			

			randVersion := "1.0"

			push := Cf("push", appName, "-p", appPath, "-k", "500M").Wait(LONG_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: " + randVersion)) 

	                Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack " + randVersion))

		})
	
		It("fails with -k parameter smaller than the real app size under 1G", func() {
			//create app with 510M	
			createDeployment(appPath, 510*1024*1024)			

			randVersion := "1.0"
			push := Cf("push", appName, "-p", appPath, "-k", "500M").Wait(LONG_TIMEOUT)
                        Expect(push).ToNot(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: " + randVersion)) 
			Expect(push).To(Say("FAILED"))
			Expect(push).To(Say("Start unsuccessful"))


		})
	
				
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())			
	        })	
	})

})
