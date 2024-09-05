package eks

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Eks Suite")
}
