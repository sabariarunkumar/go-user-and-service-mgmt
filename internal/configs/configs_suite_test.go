package configs

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestSuite...
func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Env Test Suite")
}