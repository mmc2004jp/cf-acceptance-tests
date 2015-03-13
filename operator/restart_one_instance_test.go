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

var _ = Describe("Restart One Instance", func() {
	var appName string

	BeforeEach(func() {
		appName = generator.RandomName()
	})

	AfterEach(func() {
		Expect(cf.Cf("delete", appName, "-f").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))
	})

	It("Starts one instance only and avoids downtime of entire app", func() {

		Expect(cf.Cf("push", appName, "-p", assets.NewAssets().HelloWorld, "-i", "3").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		Expect(cf.Cf("restart-app-instance", appName, "0").Wait(CF_PUSH_TIMEOUT)).To(Exit(0))

		app := cf.Cf("app", appName).Wait(DEFAULT_TIMEOUT)
		Expect(app).To(Exit(0))
		Expect(app).To(Say("#0   down"))
		Expect(app).To(Say("#1   running"))
		Expect(app).To(Say("#2   running"))

	})

})
