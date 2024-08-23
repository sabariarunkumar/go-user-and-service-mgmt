package configs

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Payload Validator", func() {

	Context("Env variables test", func() {
		BeforeEach(func() {
			os.Unsetenv("testVar")
		})
		It("already set string env variable", func() {
			os.Setenv("testVar", "value")
			res := getEnv("testVar", "")
			Expect(res).To(Equal("value"))
		})
		It("default string env variable", func() {
			res := getEnv("testVar", "defaultValue")
			Expect(res).To(Equal("defaultValue"))
		})
		It("get valid int env variable", func() {
			var valueInt int64 = 10
			os.Setenv("testVar", "10")
			res := getEnvAsInt("testVar", 0)
			Expect(res).To(Equal(valueInt))
		})
		It("get invalid int env variable", func() {
			os.Setenv("testVar", "ten")
			res := getEnvAsInt("testVar", 0)
			Expect(res).To(Equal(int64(0)))
		})
		It("default int env variable", func() {
			res := getEnvAsInt("testVar", 0)
			Expect(res).To(Equal(int64(0)))
		})
	})
	Context("Init config", func() {
		It("load valid env file", func() {
			tempFile := "temp-config.yaml"
			file, err := os.Create(tempFile)
			if err != nil {
				Fail(fmt.Sprintf("Prerequisite not met: Error creating file: %+v", err))
				return
			}
			defer func() {
				_ = os.Remove(tempFile)
				file.Close()
			}()
			content := "PORT: 80\n"
			_, err = file.WriteString(content)
			if err != nil {
				Fail(fmt.Sprintf("Prerequisite not met: Error writing to file: %+v", err))
				return
			}
			config, err := InitConfig("temp-config.yaml")
			Expect(config.ServerPort).To(Equal("80"))
			Expect(err).To(BeNil())

		})
		It("load invalid env file", func() {
			config, err := InitConfig("temp.yaml")
			Expect(config).To(BeNil())
			Expect(err.Error()).
				To(Equal("failed to load env file from configured path open temp.yaml: no such file or directory"))
		})
	})

})
