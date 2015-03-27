package operator

import (
//	"fmt"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/cf-acceptance-tests/helpers/assets"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

)

var _ = Describe("Bind Service", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	It("applies VCAP_SERVICES", func() {
		
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().HelloWorld).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		appEnv := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
		Expect(appEnv).To(Exit(0))
		Expect(appEnv.Out.Contents()).NotTo(ContainSubstring("VCAP_SERVICES"))


		serviceName := generator.RandomName()
		Expect(cf.Cf("cups", serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		//bind service, VCAP_SERVICE will be applied
		Expect(cf.Cf("bind-service", appName, serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Expect(cf.Cf("restart", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		appEnv = cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
		Expect(appEnv).To(Exit(0))
		Expect(appEnv).To(Say("VCAP_SERVICES"))

		//unbind service, VCAP_SERVICES will be cleared
		Expect(cf.Cf("unbind-service", appName, serviceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Expect(cf.Cf("restart", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		appEnv = cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
		Expect(appEnv).To(Exit(0))
		Expect(appEnv.Out.Contents()).NotTo(ContainSubstring("VCAP_SERVICES"))


	})

})
