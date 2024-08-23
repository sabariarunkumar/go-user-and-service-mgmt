package auth

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("JWT Tests", func() {

	var (
		token  *string
		secret = []byte("secret")
	)
	It("create JWT", func() {
		var err error
		token, err = CreateJWT(secret, 55, "test@gmail.com", "basic")
		Expect(err).To(BeNil())
		Expect(*token).To(Not(BeEmpty()))
	})

	Context("validate JWT", func() {
		It("validate against recently issued token ", func() {
			recvToken, err := ValidateJWT(secret, *token)
			Expect(err).To(BeNil())
			Expect(recvToken).To(Not(BeNil()))
			Expect(recvToken.Valid).To(BeTrue())
		})
		It("validate against empty token ", func() {
			_, err := ValidateJWT(secret, "")
			Expect(err).To(Not(BeNil()))
		})
	})
	Context("validate against expired token", func() {

		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImFkbWluQGtvbmcuY" +
			"29tIiwiZXhwIjoxNzIzNDY4NTgxLCJyb2xlIjoiYWRtaW4ifQ._FC85LCw0nGviDhK0EAmTWneTWUFQBJC41ivdpnH5b4"
		It("validate against empty token ", func() {
			_, err := ValidateJWT(secret, token)
			Expect(err).To(Not(BeNil()))
		})
	})
})
