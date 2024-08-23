package role

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// TestSuite...
func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Roles Test Suite")
}
