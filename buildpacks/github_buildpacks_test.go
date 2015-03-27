package buildpacks

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"
)

var _ = Describe("Buildpacks", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("ruby", func() {
		It("pushes successfully", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().RubySimple, "-b", "https://github.com/cloudfoundry/ruby-buildpack.git").Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Healthy"))
		})
	})


	Describe("node", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Node, "-b", "https://github.com/cloudfoundry/nodejs-buildpack.git", "-c", "node app.js").Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from a node app!"))
		})
	})

	Describe("golang", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Golang, "-b", "https://github.com/cloudfoundry/go-buildpack.git").Wait(LONG_TIMEOUT)).To(Exit(0))
 
			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("go, world"))
		})
	})

	Describe("php", func() {
		// This test requires more time during push, because the php buildpack is slower than your average bear
		var phpPushTimeout = LONG_TIMEOUT + 6*time.Minute

		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Php, "-b", "https://github.com/cloudfoundry/php-buildpack.git").Wait(phpPushTimeout)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello from php"))
		})
	})

	Describe("python", func() {
		It("makes the app reachable via its bound route", func() {
			Expect(cf.Cf("push", appName, "-p", assets.NewAssets().Python, "-b", "https://github.com/cloudfoundry/python-buildpack.git").Wait(LONG_TIMEOUT)).To(Exit(0))

			Eventually(func() string {
				return helpers.CurlAppRoot(appName)
			}, DEFAULT_TIMEOUT).Should(ContainSubstring("python, world"))
		})
	})


})
