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


	createManifest := func(manifestPath string, content string) {

		manifestFile, err := os.Create(path.Join(manifestPath, "manifest.yml"))
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

			createManifest(tmpdir, content)

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
			createManifest(appPath, content)

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
			createManifest(tmpdir, content)


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

			createManifest(appPath, content)

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
								
				
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())			
			err = os.RemoveAll(path.Dir(manifestFilePath))
			Expect(err).ToNot(HaveOccurred())			
	        })	
	})

})
