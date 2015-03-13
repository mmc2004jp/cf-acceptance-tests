package operator

import (
//	"fmt"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
//	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

)

var _ = Describe("Blue Green Deployment", func() {
	var blueAppName string
	var greenAppName string

	BeforeEach(func() {
		blueAppName = generator.RandomName()
		greenAppName = generator.RandomName()
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", blueAppName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Expect(cf.Cf("delete", greenAppName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	It("avoids the downtime to upgrade the application", func() {

		//1. push an app
		Expect(cf.Cf("push", blueAppName, "-p", assets.NewAssets().HelloWorld).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(blueAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, world!"))
		
		//2. update app and push. Here, using a different app for testing
		Expect(cf.Cf("push", greenAppName, "-p", assets.NewAssets().Dora).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		Eventually(func() string {
			return helpers.CurlAppRoot(greenAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		
		//3. map original route to Green	
		Expect(cf.Cf("map-route", greenAppName, helpers.LoadConfig().AppsDomain, "-n", blueAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		//it should response from the Green app when accessing GREEN domain
		Eventually(func() string {
			return helpers.CurlAppRoot(greenAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))

		//it should response from both the Green app and Blue app when accessing the BLUE domain
		Eventually(func() string {
			return helpers.CurlAppRoot(blueAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		Eventually(func() string {
			return helpers.CurlAppRoot(blueAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hello, world!"))
		
		//4. unmap route to Blue
		Expect(cf.Cf("unmap-route", blueAppName, helpers.LoadConfig().AppsDomain, "-n", blueAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		
		//it should response from the Green app when accessing GREEN domain
		Eventually(func() string {
			return helpers.CurlAppRoot(greenAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		
		//it should response from the Green app when accessing the BLUE domain
		Eventually(func() string {
			return helpers.CurlAppRoot(blueAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		//it should NOT response from the Blue app when accessing the BLUE domain
		Eventually(func() string {
			return helpers.CurlAppRoot(blueAppName)
		}, DEFAULT_TIMEOUT).ShouldNot(ContainSubstring("Hello, world!"))
		
		//5. remove temporary route to Green
		Expect(cf.Cf("unmap-route", greenAppName, helpers.LoadConfig().AppsDomain, "-n", greenAppName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		
		//it should response only from the Green app when accessing the BLUE domain
		Eventually(func() string {
			return helpers.CurlAppRoot(blueAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("Hi, I'm Dora!"))
		//it should NOT response from the Green app when accessing the GREEN domain
		Eventually(func() string {
			return helpers.CurlAppRoot(greenAppName)
		}, DEFAULT_TIMEOUT).Should(ContainSubstring("404 Not Found"))
		

	})

})
