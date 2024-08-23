package auth

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Password Tests", func() {

	var (
		password       = "password"
		hashedPassword string
	)

	It("Generate valid Password Hash", func() {
		var err error
		hashedPassword, err = GeneratePasswordHash(password)
		Expect(err).To(BeNil())
		Expect(hashedPassword).To(Not(BeEmpty()))
	})
	It("password of higher length not accepted by bcrypt", func() {
		passwordOfHigherLength := "Lorem ipsum dolor sit amet," +
			" consectetuer adipiscing elit. Aenean commodo ligula eget dolor. Aenean m"
		hashedPasswordNotExpected, err := GeneratePasswordHash(passwordOfHigherLength)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("password length exceeds 72 bytes"))
		Expect(hashedPasswordNotExpected).To(BeEmpty())
	})
	It("Compare Hash And Password", func() {
		result := CompareHashAndPassword(hashedPassword, []byte(password))
		Expect(result).To(BeTrue())
	})
	Context("Generate Password", func() {
		It("check if generated password is of tempPassLength", func() {
			genPassword, err := GeneratePassword()
			Expect(err).To(BeNil())
			Expect(genPassword).To(Not(BeNil()))
			Expect(*genPassword).To(HaveLen(12))
		})
	})

})
