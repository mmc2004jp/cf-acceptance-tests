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
  web: while true; do { echo -e 'HTTP/1.1 200 OK\r\n';echo "hi from a simple admin buildpack $buildpackVersion."; echo "HOME:\$HOME"; echo "MEMORY_LIMIT:\$MEMORY_LIMIT"; echo "PORT:\$PORT";echo "PWD:\$PWD"; echo "TMPDIR:\$TMPDIR"; echo "USER:\$USER"; echo "VCAP_APP_HOST:\$VCAP_APP_HOST"; echo "VCAP_APPLICATION:\$VCAP_APPLICATION"; echo "VCAP_APP_PORT:\$VCAP_APP_PORT"; echo "VCAP_SERVICES:\$VCAP_SERVICES"; } | nc -l \$PORT; done
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


	Context("when it prints environment variables", func() {

               It("completes successfully", func() {
                        randVersion := "1.0"
                        createBuildPack(BuildpackName, randVersion, 1)

                        content := fmt.Sprintf(`
---
applications:
- name: %s
`, appName)
                        createDeployment(appPath, content)

                        Expect(Cf("push", appName, "-p", appPath, "--no-start", "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

	                serviceName := RandomName()
        	        Expect(Cf("cups", serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

                	//bind service, VCAP_SERVICE will be applied
	                Expect(Cf("bind-service", appName, serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

        	        Expect(Cf("start", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))


			var curlResponse string
			Eventually(func() string {
				curlResponse = helpers.CurlAppRoot(appName)
				return curlResponse
                        }, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack"))

			Expect(curlResponse).To(ContainSubstring("HOME:/home/vcap/app"))
			Expect(curlResponse).To(ContainSubstring("MEMORY_LIMIT:512m"))
			Expect(curlResponse).To(MatchRegexp("PORT:[0-9]+"))
			Expect(curlResponse).To(ContainSubstring("PWD:/home/vcap/app"))
			Expect(curlResponse).To(ContainSubstring("TMPDIR:/home/vcap/tmp"))
			Expect(curlResponse).To(ContainSubstring("USER:vcap"))
			Expect(curlResponse).To(ContainSubstring("VCAP_APP_HOST:0.0.0.0"))
			Expect(curlResponse).To(MatchRegexp("VCAP_APPLICATION:{.+}"))
			Expect(curlResponse).To(MatchRegexp("VCAP_APP_PORT:[0-9]+"))
			Expect(curlResponse).To(MatchRegexp("VCAP_SERVICES:{.+}"))
 
                })

		
		AfterEach(func() {
	                deleteBuildPack(BuildpackName)
			Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			err := os.RemoveAll(appPath)
			Expect(err).ToNot(HaveOccurred())			
	        })	
	})

})
