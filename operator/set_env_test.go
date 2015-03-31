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

	It("applies environment", func() {
		
		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().HelloWorld).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
		appEnv := cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
		Expect(appEnv).To(Exit(0))
		Expect(appEnv.Out.Contents()).NotTo(ContainSubstring("APP_ENV_1"))



		//bind service, VCAP_SERVICE will be applied
		Expect(cf.Cf("set-env", appName, "APP_ENV_1", "app env 1").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		Expect(cf.Cf("restart", appName).Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		appEnv = cf.Cf("env", appName).Wait(DEFAULT_TIMEOUT)
		Expect(appEnv).To(Exit(0))
		Expect(appEnv).To(Say("APP_ENV_1: app env 1"))

	})

})
