package aks

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aks Suite")
}
