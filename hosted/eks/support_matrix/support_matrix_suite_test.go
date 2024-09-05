package support_matrix_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/rancher/hosted-providers-e2e/hosted/helpers"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/kubernetesversions"
)

var (
	availableVersionList []string
	ctx                  helpers.Context
)

func TestSupportMatrix(t *testing.T) {
	RegisterFailHandler(Fail)
	var err error
	ctx, err = helpers.CommonBeforeSuite("eks")
	Expect(err).To(BeNil())
	availableVersionList, err = kubernetesversions.ListEKSAllVersions(ctx.RancherClient)
	Expect(err).To(BeNil())
	Expect(availableVersionList).ToNot(BeEmpty())
	RunSpecs(t, "SupportMatrix Suite")
}
