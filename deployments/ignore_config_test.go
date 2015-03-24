package deployments

import (
	"fmt"
	"io/ioutil"
	"os"
	"io"
	"strings"
	"archive/zip"
	"path"
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

		appPath string
//		cfignoreFilePath string

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
		_, err = os.Create(path.Join(appPath, "RequestUri.jsp"))
		Expect(err).ToNot(HaveOccurred())


		err = os.Mkdir(path.Join(appPath, "dir1"), 0644)
		Expect(err).ToNot(HaveOccurred())
		err = os.Mkdir(path.Join(appPath, "dir2"), 0644)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Create(path.Join(appPath, "README.txt~"))
		Expect(err).ToNot(HaveOccurred())

	}


/*
approot/
  dir/
    manifest.yml
    .gitignore
    .git
    .hg
    .svn
    _darcs
    .DS_Store
  manifest.yml
  .gitignore
  .git
  .hg
  .svn
  _darcs
  .DS_Store
  and some other files
*/

	createDeploymentWithIgnoredFiles := func(appPath string) {

		_, err := os.Create(path.Join(appPath, matchingFilename(appName)))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, "manifest.yml"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, ".gitignore"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, ".git"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, ".hg"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, ".svn"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, "_darcs"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, ".DS_Store"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(appPath, "some-file"))
		Expect(err).ToNot(HaveOccurred())


		dirPath := path.Join(appPath, "dir")
		err = os.Mkdir(dirPath, 0775)
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(dirPath, "manifest.yml"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(dirPath, ".gitignore"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(dirPath, ".git"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(dirPath, ".hg"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(dirPath, ".svn"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(dirPath, "_darcs"))
		Expect(err).ToNot(HaveOccurred())

		_, err = os.Create(path.Join(dirPath, ".DS_Store"))
		Expect(err).ToNot(HaveOccurred())


	}


	addCfIgnoreFile := func(appPath string) {

		cfignoreFile, err := os.Create(path.Join(appPath, ".cfignore"))
		Expect(err).ToNot(HaveOccurred())

		_, err = cfignoreFile.WriteString(
		fmt.Sprintf(`
dir1/
dir2/
*~
` ))

		Expect(err).ToNot(HaveOccurred())

	}


       	addCfIgnoreFileFromZip := func(cfIgnoreFileName string, appPath string) {

                cfIgnoreFile, err := os.Create(path.Join(appPath, ".cfignore"))
                Expect(err).ToNot(HaveOccurred())

                // Open the zip archive for reading.
                cfIgnoreZipFiles := assets.NewAssets().CfIgnoreFiles

                r, err := zip.OpenReader(cfIgnoreZipFiles)
		defer r.Close()
                Expect(err).ToNot(HaveOccurred())


                // Iterate through the files in the archive,
                // until it finds the specified ignore file
                for _, f := range r.File {
                        if strings.EqualFold(cfIgnoreFileName, path.Base(f.Name)) {
                                rc, err := f.Open()
                                Expect(err).ToNot(HaveOccurred())

                                defer rc.Close()

                                _, err = io.Copy(cfIgnoreFile, rc)
                                Expect(err).ToNot(HaveOccurred())

                                break
                        }
                }

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

	Context("when it has .cfignore", func() {
	
		It("can ignore the directories specified in .cfignore", func() {

			createDeployment(appPath)

			push := Cf("push", appName, "-p", appPath, "-m", "128M" ).Wait(CF_PUSH_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: 1.0")) 

                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			files := Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
			Expect(files).To(Exit(0))
			Expect(files).To(Say("dir1/"))
			Expect(files).To(Say("dir2/"))


			//add .cfignore file and push again
			addCfIgnoreFile(appPath)
			push = Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


			files = Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
			Expect(files).To(Exit(0))
			Expect(files).NotTo(Say("dir1/"))
			Expect(files).NotTo(Say("dir2/"))


		})
									
		It("can ignore the files specified in .cfignore", func() {

			createDeployment(appPath)

			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: 1.0")) 

                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))

			files := Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
			Expect(files).To(Exit(0))
			Expect(files).To(Say("README.txt~"))


			//add .cfignore file and push again
			addCfIgnoreFile(appPath)
			push = Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


			files = Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
			Expect(files).To(Exit(0))
			Expect(files).NotTo(Say("README.txt~"))
		

		})
	
	        AfterEach(func() {
        	        deleteBuildPack(BuildpackName)
	                Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
        	        err := os.RemoveAll(appPath)
	                Expect(err).ToNot(HaveOccurred())
        	})


	})

	Context("when .cfignore has non-readable characters in Shift_JIS encoding", func() {
		
		It("completes successfully if the first line is readable", func() {

			javaAppPath := assets.NewAssets().Java

                        //add .cfignore file 
                        addCfIgnoreFileFromZip("cfignore_readable_first_line.Shift_JIS", javaAppPath)


			Expect(Cf("push", appName, "-p", javaAppPath, "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
                        Eventually(func() string {
                                 return helpers.CurlAppRoot(appName)
                        }, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))


                        files := Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
                        Expect(files).To(Exit(0))
			Expect(files).NotTo(Say("RequestUri.jsp"))
			Expect(files).To(Say("Ω.jsp"))


                })				
	
		It("completes successfully to filter out other files if the first line is non-readable", func() {

			javaAppPath := assets.NewAssets().Java

                        //add .cfignore file and push again
                        addCfIgnoreFileFromZip("cfignore_nonreadable_first_line.Shift_JIS", javaAppPath)


                        Expect(Cf("push", appName, "-p", javaAppPath, "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
                        Eventually(func() string {
                                 return helpers.CurlAppRoot(appName)
                        }, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))


                        files := Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
                        Expect(files).To(Exit(0))
                        Expect(files).NotTo(Say("RequestUri.jsp"))
			Expect(files).To(Say("Ω.jsp"))


                })				
		
	        AfterEach(func() {
        	        deleteBuildPack(BuildpackName)
	                Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
         	        err := os.RemoveAll(appPath)
	                Expect(err).ToNot(HaveOccurred())
        	        err = os.Remove(path.Join(assets.NewAssets().Java, ".cfignore"))
	                Expect(err).ToNot(HaveOccurred())
        	})

		
	})

	Context("when .cfignore has non-readable characters in UTF-8 encoding", func() {
		
		It("completes successfully if the first line is readable", func() {

			javaAppPath := assets.NewAssets().Java

                        //add .cfignore file 
                        addCfIgnoreFileFromZip("cfignore_readable_first_line.UTF-8", javaAppPath)


			Expect(Cf("push", appName, "-p", javaAppPath, "-m", "512M").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
                        Eventually(func() string {
                                 return helpers.CurlAppRoot(appName)
                        }, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, from your friendly neighborhood Java JSP!"))


                        files := Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
                        Expect(files).To(Exit(0))
                        Expect(files).NotTo(Say("RequestUri.jsp"))
			Expect(files).NotTo(Say("Ω.jsp"))



                })				
	
	
	        AfterEach(func() {
        	        deleteBuildPack(BuildpackName)
	                Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
        	        err := os.RemoveAll(appPath)
	                Expect(err).ToNot(HaveOccurred())
         	        err = os.Remove(path.Join(assets.NewAssets().Java, ".cfignore"))
	                Expect(err).ToNot(HaveOccurred())
 	      	})

		
	})


	Context("when it has default ignored files", func() {
	
		It("ignores them by default, like .gitignore, .git, etc", func() {
			createDeploymentWithIgnoredFiles(appPath)

			push := Cf("push", appName, "-p", appPath, "-m", "128M").Wait(CF_PUSH_TIMEOUT)
                        Expect(push).To(Exit(0))
			Expect(push).To(Say("Staging with Simple Buildpack"))
                 	Expect(push).To(Say("VERSION: 1.0")) 

                        Eventually(func() string {
			         return helpers.CurlAppRoot(appName)
                  	}, DEFAULT_TIMEOUT).Should(ContainSubstring("hi from a simple admin buildpack 1.0"))


			files := Cf("files", appName, "app").Wait(DEFAULT_TIMEOUT)
			Expect(files).To(Exit(0))
			Expect(files).NotTo(Say("manifest.yml"))
			Expect(files).NotTo(Say(".gitignore"))
			Expect(files).NotTo(Say(".git"))
			Expect(files).NotTo(Say(".hg"))
			Expect(files).NotTo(Say(".svn"))
			Expect(files).NotTo(Say("_darcs"))
			Expect(files).NotTo(Say(".DS_Store"))
			Expect(files).NotTo(Say(".gitignore"))
			Expect(files).NotTo(Say(".gitignore"))
		
			files = Cf("files", appName, "app/dir").Wait(DEFAULT_TIMEOUT)
			Expect(files).To(Exit(0))
			Expect(files).To(Say("manifest.yml"))
			Expect(files).NotTo(Say(".gitignore"))
			Expect(files).NotTo(Say(".git"))
			Expect(files).NotTo(Say(".hg"))
			Expect(files).NotTo(Say(".svn"))
			Expect(files).NotTo(Say("_darcs"))
			Expect(files).NotTo(Say(".DS_Store"))
			Expect(files).NotTo(Say(".gitignore"))
			Expect(files).NotTo(Say(".gitignore"))
	
		})
		
	        AfterEach(func() {
        	        deleteBuildPack(BuildpackName)
	                Expect(Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
        	        err := os.RemoveAll(appPath)
	                Expect(err).ToNot(HaveOccurred())
        	})

			
	})


})
