package helpers

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/gomega"
	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials/aws"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials/azure"
	"github.com/rancher/rancher/tests/framework/extensions/cloudcredentials/google"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/pipeline"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/rancher/rancher/tests/framework/pkg/wait"
	"github.com/rancher/rancher/tests/v2prov/defaults"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Timeout = 30 * time.Minute
)

var (
	rancherPassword = os.Getenv("RANCHER_PASSWORD")
	rancherHostname = os.Getenv("RANCHER_HOSTNAME")
	cloudCredential *cloudcredentials.CloudCredential
	rancherConfig   *rancher.Config
)

type Context struct {
	CloudCred     *cloudcredentials.CloudCredential
	RancherClient *rancher.Client
	Session       *session.Session
}

func CommonBeforeSuite(cloud string) (Context, error) {

	rancherConfig := new(rancher.Config)
	Eventually(rancherConfig, "10s").ShouldNot(BeNil())

	config.LoadAndUpdateConfig(rancher.ConfigurationFileKey, rancherConfig, func() {
		rancherConfig.Host = rancherHostname
	})

	token, err := pipeline.CreateAdminToken(rancherPassword, rancherConfig)
	Expect(err).To(BeNil())

	config.LoadAndUpdateConfig(rancher.ConfigurationFileKey, rancherConfig, func() {
		rancherConfig.AdminToken = token
	})

	testSession := session.NewSession()
	rancherClient, err := rancher.NewClient(rancherConfig.AdminToken, testSession)
	Expect(err).To(BeNil())

	setting := new(management.Setting)
	resp, err := rancherClient.Management.Setting.ByID("server-url")
	Expect(err).To(BeNil())

	setting.Source = "env"
	setting.Value = fmt.Sprintf("https://%s", rancherHostname)
	resp, err = rancherClient.Management.Setting.Update(resp, setting)
	Expect(err).To(BeNil())

	switch cloud {
	case "aks":
		credentialConfig := new(cloudcredentials.AzureCredentialConfig)
		config.LoadAndUpdateConfig("azureCredentials", credentialConfig, func() {
			credentialConfig.ClientID = os.Getenv("AKS_CLIENT_ID")
			credentialConfig.SubscriptionID = os.Getenv("AKS_SUBSCRIPTION_ID")
			credentialConfig.ClientSecret = os.Getenv("AKS_CLIENT_SECRET")
		})
		cloudCredential, err = azure.CreateAzureCloudCredentials(rancherClient)
		Expect(err).To(BeNil())
	case "eks":
		credentialConfig := new(cloudcredentials.AmazonEC2CredentialConfig)
		config.LoadAndUpdateConfig("awsCredentials", credentialConfig, func() {
			credentialConfig.AccessKey = os.Getenv("AWS_ACCESS_KEY_ID")
			credentialConfig.SecretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
			credentialConfig.DefaultRegion = os.Getenv("EKS_REGION")
		})
		cloudCredential, err = aws.CreateAWSCloudCredentials(rancherClient)
		Expect(err).To(BeNil())
	case "gke":
		credentialConfig := new(cloudcredentials.GoogleCredentialConfig)
		config.LoadAndUpdateConfig("googleCredentials", credentialConfig, func() {
			credentialConfig.AuthEncodedJSON = os.Getenv("GCP_CREDENTIALS")
		})
		cloudCredential, err = google.CreateGoogleCloudCredentials(rancherClient)
		Expect(err).To(BeNil())
	}

	return Context{
		CloudCred:     cloudCredential,
		RancherClient: rancherClient,
		Session:       testSession,
	}, nil
}

// WaitUntilClusterIsReady waits until the cluster is in a Ready state,
// fetch the cluster again once it's ready so that it has everything up to date and then return it.
// For e.g. once the cluster has been updated, it contains information such as Version.GitVersion which it does not have before it's ready
func WaitUntilClusterIsReady(cluster *management.Cluster, client *rancher.Client) (*management.Cluster, error) {
	opts := metav1.ListOptions{FieldSelector: "metadata.name=" + cluster.ID, TimeoutSeconds: &defaults.WatchTimeoutSeconds}
	watchInterface, err := client.GetManagementWatchInterface(management.ClusterType, opts)
	if err != nil {
		return nil, err
	}

	watchFunc := clusters.IsHostedProvisioningClusterReady

	err = wait.WatchWait(watchInterface, watchFunc)
	if err != nil {
		return nil, err
	}
	return client.Management.Cluster.ByID(cluster.ID)
}
