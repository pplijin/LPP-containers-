package p0_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
	namegen "github.com/rancher/rancher/tests/framework/pkg/namegenerator"
)

var (
	ctx         helpers.Context
	clusterName string
	zone        = "us-central1-c"
	project     = os.Getenv("GKE_PROJECT_ID")
	k8sVersion  = "1.27.4-gke.900"
	increaseBy  = 1
)

func TestP0(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "P0 Suite")
}

var _ = BeforeEach(func() {
	var err error
	ctx, err = helpers.CommonBeforeSuite("gke")
	Expect(err).To(BeNil())
	clusterName = namegen.AppendRandomString("gkehostcluster")
})
