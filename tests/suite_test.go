package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSuite(t *testing.T) {
	suiteConfig, reportConfig := GinkgoConfiguration()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Suites", suiteConfig, reportConfig)
}
