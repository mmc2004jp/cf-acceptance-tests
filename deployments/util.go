package deployments

import (
	"fmt"
	"os"
	"log"
	"os/exec"
	"io"
	"bytes"
	"strings"
	"io/ioutil"
	"encoding/json"
	"path"

//	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	. "github.com/onsi/gomega"
	"github.com/cloudfoundry-incubator/cf-test-helpers/runner"
//	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	archive_helpers "github.com/pivotal-golang/archiver/extractor/test_helper"	
)



type AppResource struct {
	Entity struct {
		Name string `json:"name"`
		DetectedBuildpack string `json:"detected_buildpack"`
	} `json:"entity"`
}

type BuildpackResource struct {
	Entity struct {
		Name string `json:"name"`
		Position int `json:"position"`
		Enabled bool `json:"enabled"`
		Locked bool `json:"locked"`
		Filename string `json:"filename"`
	} `json:"entity"`
}

type AppsResponse struct {
	TotalResults int `json:"total_results"`
	Resources []AppResource `json:"resources"`
}

type BuildpacksResponse struct {
	TotalResults int `json:"total_results"`
	Resources []BuildpackResource `json:"resources"`
}

type ErrCodeSet struct {
	StatusCode string
	ErrorCode string
}

func getBuildpacks() (response BuildpacksResponse) {
	json.Unmarshal(execCurl("/v2/buildpacks"), &response)
	return response
}

func getAppByName(appName string) (response AppsResponse) {
	json.Unmarshal(execCurl(fmt.Sprintf("/v2/apps?q=name:%s", appName)), &response)
	return response
}

func execCurl(request string) ([]byte) {
	return Cf("curl", request).Wait(DEFAULT_TIMEOUT).Out.Contents()
}

func ensureAppNotRegistered(appName string) {
	response := getAppByName(appName)

	if response.TotalResults <= 0 {
		return
	}

	Expect(response.Resources[0].Entity.Name).To(Equal(appName))
}

func ensureBuildpacksNotRegistered(buildpackNames []string) {
	response := getBuildpacks()

	if response.TotalResults <= 0 {
		return
	}

	for _, resource := range response.Resources {
		for _, buildpackName := range buildpackNames {
			Expect(resource.Entity.Name).ShouldNot(Equal(buildpackName))
		}
	}
}

func AsInterceptCommand(stdinString string, env map[string]string, actions func()){
	originalCommandIntercepter := runner.CommandInterceptor
	runner.CommandInterceptor = func(cmd *exec.Cmd) *exec.Cmd {
		if stdinString != "" {
			setStdin(cmd, stdinString)
		}
		if env != nil {
			setEnv(cmd, env)
		}
		return cmd
	}
	defer resetCommandIntercepter(originalCommandIntercepter)
	actions()
}

func resetCommandIntercepter(originalCommandIntercepter func(cmd *exec.Cmd) *exec.Cmd){
	runner.CommandInterceptor = originalCommandIntercepter
}

func setStdin(cmd *exec.Cmd, stdinString string){
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Panic(err)
	}
	defer stdin.Close()
	io.Copy(stdin, bytes.NewBufferString(stdinString))
}

func setEnv(cmd *exec.Cmd, newEnv map[string]string){
	orgEnv := os.Environ()
	mergedEnvMap := make(map[string]string)

	for _, v := range orgEnv {
		variables := strings.SplitN(v, "=", 2)
		orgKey := variables[0]
		orgValue := variables[1]
		mergedEnvMap[orgKey] = orgValue
	}

	for k, v := range newEnv {
		mergedEnvMap[k] = v
	}

	for k, v := range mergedEnvMap {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
}


func matchingFilename(appName string) string {
		return fmt.Sprintf("stack-match-%s", appName)
}

func CreateZipArchive(buildpackArchivePath string, appName string, version string, timeout int64) { 
	archive_helpers.CreateZipArchive(buildpackArchivePath, []archive_helpers.ArchiveFile{
			{
				Name: "bin/compile",
				Body: fmt.Sprintf(`#!/usr/bin/env bash

					sleep 1 # give loggregator time to start streaming the logs

					echo "Staging with Simple Buildpack"
					echo "VERSION: %s" 
					echo "Sleeping %ds..."
					sleep %d 
					echo "Wake up...."

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


func CreateBuildPack(buildPackName string, appName string, version string, timeout int64) {		
	AsUser(context.AdminUserContext(), func() {
		var err error
		var tmpdir string

		tmpdir, err = ioutil.TempDir(os.TempDir(), "matching-buildpack")
		Expect(err).ToNot(HaveOccurred())

		buildpackArchivePath := path.Join(tmpdir, "buildpack_" + version + ".zip")

		CreateZipArchive(buildpackArchivePath, appName, version, timeout)

		createBuildpack := Cf("create-buildpack", buildPackName, buildpackArchivePath, "0").Wait(DEFAULT_TIMEOUT)
		Expect(createBuildpack).Should(Exit(0))
		Expect(createBuildpack).Should(Say("Creating"))
		Expect(createBuildpack).Should(Say("OK"))
		Expect(createBuildpack).Should(Say("Uploading"))
		Expect(createBuildpack).Should(Say("OK"))

		//clean the temporary directory of the buildpack
		err = os.RemoveAll(tmpdir)
		Expect(err).ToNot(HaveOccurred())			
	})
}
	
func DeleteBuildPack(buildpackName string) { 
		
	AsUser(context.AdminUserContext(), func() {
		Expect(Cf("delete-buildpack", buildpackName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
}

func CreateDeployment (appPath string, appName string, size int64)  {

		_, err := os.Create(path.Join(appPath, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())
		appFile, err := os.Create(path.Join(appPath, "some-file"))
		Expect(err).ToNot(HaveOccurred())
		err = appFile.Truncate(size)
		Expect(err).ToNot(HaveOccurred())

}


func CreateManifest(manifestPath string, manifestFileName string, content string) string {

	manifestFile, err := os.Create(path.Join(manifestPath, manifestFileName))
	Expect(err).ToNot(HaveOccurred())

	_, err = manifestFile.WriteString(content)
	Expect(err).ToNot(HaveOccurred())

	return manifestFile.Name()

}

