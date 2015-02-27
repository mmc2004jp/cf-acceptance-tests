package space

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gbytes"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
)


var _ = Describe("Create Space", func() {
		
	var orgName string

	BeforeEach(func() {
		orgName = "ORG-" + generator.RandomName()
		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("create-org", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

	})

	AfterEach(func() {
		cf.AsUser(context.AdminUserContext(), func() {
			Expect(cf.Cf("delete-org", orgName, "-f").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		})

	})

	Describe("create space without specifying org", func() {
		var spaceName = "SPACE-" + generator.RandomName()
		It("completes successfully without specifying org", func() {
			cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("target", "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	         		Expect(cf.Cf("create-space", spaceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		})

	})

	Describe("create space named 'name with space'", func() {
		It("completes successfully with name having space", func() {
			cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("target", "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	         		Expect(cf.Cf("create-space", "name with space").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		})

	})



	Describe("create space under the specified org", func() {
		var spaceName = "SPACE-" + generator.RandomName()
		It("completes successfully", func() {
			cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		})

		It("fails to create duplicated space", func() {
			cf.AsUser(context.AdminUserContext(), func() {
				Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("create-space", spaceName, "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Say("already exists"))
			})
		})
	})

	Describe("create space with a specified space quota", func() {
		var quotaName = "QUOTA-" + generator.RandomName()
		It("completes successfully with order -q quota -o org", func() {
			var spaceName = "SPACE-" + generator.RandomName()
	        	cf.AsUser(context.AdminUserContext(), func() {
         			Expect(cf.Cf("target", "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("create-space-quota", quotaName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("create-space", spaceName, "-q", quotaName, "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		})

		It("completes successfully with order -o org -q quota", func() {
			var spaceName = "SPACE-" + generator.RandomName()
			cf.AsUser(context.AdminUserContext(), func() {
         			Expect(cf.Cf("target", "-o", orgName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("create-space-quota", quotaName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
				Expect(cf.Cf("create-space", spaceName, "-o", orgName, "-q", quotaName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
			})
		})
	})

})

