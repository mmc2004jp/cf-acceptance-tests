package internet_dependent_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"time"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("GitBuildpack", func() {
	var (
		appName string
	)

	It("uses a buildpack from a git url", func() {
		appName = generator.RandomName()
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Node, "-c", "node app.js", "-b", "https://github.com/cloudfoundry/nodejs-buildpack.git").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a node app!"))
	})

	It("pushes ruby application successfully with a buildpack from github", func() {
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().RubySimple, "-b", "https://github.com/cloudfoundry/ruby-buildpack.git").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Healthy"))
	})

	It("pushes go application successfully with a buildpack from github", func() {
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Golang, "-b", "https://github.com/cloudfoundry/go-buildpack.git").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
	})

	It("pushes php application successfully with a buildpack from github", func() {
	 
		// This test requires more time during push, because the php buildpack is slower than your average bear
		var phpPushTimeout = CF_PUSH_TIMEOUT + 6*time.Minute

		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Php, "-b", "https://github.com/cloudfoundry/php-buildpack.git").Wait(phpPushTimeout)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from php"))
	})

	It("pushes python application successfully with a buildpack from github", func() {
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Python, "-b", "https://github.com/cloudfoundry/python-buildpack.git").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Eventually(func() string {
			return helpers.CurlAppRoot(appName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("python, world"))
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})
})
