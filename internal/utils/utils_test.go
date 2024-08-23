package utils

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Utils Suite Test", func() {
	fieldTypeMap := FieldTypeBinder{
		"name": reflect.TypeOf(""),
		"age":  reflect.TypeOf(0),
	}
	Context("Ensure Fields Strictly Exists for the assumed payload", func() {

		It("when the input map has the exact fields and types as the  fieldTypeMap", func() {
			input := map[string]interface{}{
				"name": "John",
				"age":  30,
			}
			exists := EnsureFieldsStrictlyExists(input, fieldTypeMap)
			Expect(exists).To(BeTrue())
		})

		It("when the input map has lesser number of fields and types compared to the fieldTypeMap", func() {
			input := map[string]interface{}{
				"name": "John",
			}

			exists := EnsureFieldsStrictlyExists(input, fieldTypeMap)
			Expect(exists).To(BeFalse())
		})

		It("when the input map has a fields Type different compared to the fieldTypeMap", func() {
			input := map[string]interface{}{
				"name": 1,
				"age":  30,
			}

			exists := EnsureFieldsStrictlyExists(input, fieldTypeMap)
			Expect(exists).To(BeFalse())
		})

		It("when the input map has same number of field but doesn't exist in fieldTypeMap", func() {
			input := map[string]interface{}{
				"email": "test@gmail.com",
				"age":   30,
			}
			exists := EnsureFieldsStrictlyExists(input, fieldTypeMap)
			Expect(exists).To(BeFalse())
		})

		It("when the input map has more number of fields and types compared to the fieldTypeMap", func() {
			input := map[string]interface{}{
				"name":  "John",
				"age":   44,
				"email": "john@gmail.com",
			}
			exists := EnsureFieldsStrictlyExists(input, fieldTypeMap)
			Expect(exists).To(BeFalse())
		})
	})

	Context("Convert field and its type to string", func() {
		It("given set of fields and types, expected to get converted into desired format", func() {
			convertedFields := ConvertFieldTypeToString(fieldTypeMap)
			Expect(convertedFields).To(SatisfyAny(Equal("age[int],name[string]"), Equal("name[string],age[int]")))
		})
	})

	Context("Response formatting", func() {
		It("Format Generic Response", func() {
			data := "This is a generic message"
			expected := GenericResponse{Message: data}
			result := FormatGenericResponse(data)
			Expect(result).To(Equal(expected))
		})
		It("Format Error Response", func() {
			data := "This is an error message"
			expected := ErrorResponse{Error: data}
			result := FormatErrorResponse(data)
			Expect(result).To(Equal(expected))
		})
		It("Format Token Response", func() {
			token := "sample-token"
			passChangeRequired := true
			expected := TokenResponse{
				AccessToken:        token,
				PassChangeRequired: passChangeRequired,
			}
			result := FormatTokenResponse(token, passChangeRequired)
			Expect(result).To(Equal(expected))
		})
		It("Format Temp Pass Response", func() {
			pass := "temporary-pass"
			expected := TempPassResponse{TempPass: pass}
			result := FormatTempPassResponse(pass)
			Expect(result).To(Equal(expected))
		})
	})
})
