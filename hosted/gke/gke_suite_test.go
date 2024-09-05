package gke

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGke(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gke Suite")
}
