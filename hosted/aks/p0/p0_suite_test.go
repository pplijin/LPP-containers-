package p0_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
	namegen "github.com/rancher/rancher/tests/framework/pkg/namegenerator"
)

var (
	ctx         helpers.Context
	clusterName string
	location    = "eastus"
	k8sVersion  = "1.26.6"
	increaseBy  = 1
)

func TestP0(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "P0 Suite")
}

var _ = BeforeEach(func() {
	var err error
	ctx, err = helpers.CommonBeforeSuite("aks")
	Expect(err).To(BeNil())
	clusterName = namegen.AppendRandomString("akshostcluster")
})
