package setup

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSuite(t *testing.T) {
	suiteConfig, reportConfig := GinkgoConfiguration()
	RegisterFailHandler(Fail)
	RunSpecs(t, "Environment Setup Suites", suiteConfig, reportConfig)
}
