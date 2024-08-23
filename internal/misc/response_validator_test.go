package misc

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Payload Validator", func() {

	BeforeEach(func() {
		InitPayloadValidator()
	})

	It("should initialize the PayloadValidator", func() {
		Expect(PayloadValidator).NotTo(BeNil(), "PayloadValidator should be initialized")
	})

})
