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
	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"
)

var _ = Describe("Deploy Apps", func() {
	var (
		appName       string
		BuildpackName string

		appPath string
		appName1 string
		appName2 string
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

		manifestFile, err := os.Create(path.Join(appPath, "manifest.yml"))
		Expect(err).ToNot(HaveOccurred())
		manifestFilePath = manifestFile.Name()

		_, err = manifestFile.WriteString(
		fmt.Sprintf(`
---
applications:
- name: %s
  path: ./%s
- name: %s
  path: ./%s
`, appName1, appName1, appName2, appName2))

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
			//create deployment with multiple apps in one  manifest
			createDeployment(appPath)
	
		})
	})


	Context("when it specifies manifest.yml", func() {

		It("deploys multiple apps within one manifest file", func() {
			createBuildPack(BuildpackName, "1.0")	

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
	                deleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName1, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(Cf("delete", appName2, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())			
	        })	
	})

})
