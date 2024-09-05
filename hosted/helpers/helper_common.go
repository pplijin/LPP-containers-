package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/onsi/ginkgo/v2"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher/shepherd/extensions/pipeline"

	. "github.com/onsi/gomega"
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/cloudcredentials"
	"github.com/rancher/shepherd/extensions/cloudcredentials/aws"
	"github.com/rancher/shepherd/extensions/cloudcredentials/azure"
	"github.com/rancher/shepherd/extensions/cloudcredentials/google"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/shepherd/pkg/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CommonBeforeSuite(cloud string) (Context, error) {

	rancherConfig := new(rancher.Config)
	Eventually(rancherConfig, "10s").ShouldNot(BeNil())

	config.LoadAndUpdateConfig(rancher.ConfigurationFileKey, rancherConfig, func() {
		rancherConfig.Host = RancherHostname
	})

	token, err := pipeline.CreateAdminToken(RancherPassword, rancherConfig)
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
	setting.Value = fmt.Sprintf("https://%s", RancherHostname)
	resp, err = rancherClient.Management.Setting.Update(resp, setting)
	Expect(err).To(BeNil())

	var cloudCredential *cloudcredentials.CloudCredential
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
			credentialConfig.DefaultRegion = GetEKSRegion()
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

	clusterCleanup, _ := strconv.ParseBool(os.Getenv("DOWNSTREAM_CLUSTER_CLEANUP"))
	return Context{
		CloudCred:      cloudCredential,
		RancherClient:  rancherClient,
		Session:        testSession,
		ClusterCleanup: clusterCleanup,
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

// GetGKEZone fetches the value of GKE zone;
// it first obtains the value from env var GKE_ZONE, if the value is empty, it fetches the information from config file(cattle_config-import.yaml/cattle_config-provisioning.yaml)
// if none of the sources can provide a value, it returns the default value
func GetGKEZone() string {
	zone := os.Getenv("GKE_ZONE")
	if zone == "" {
		gkeConfig := new(management.GKEClusterConfigSpec)
		config.LoadConfig("gkeClusterConfig", gkeConfig)
		if gkeConfig.Zone != "" {
			zone = gkeConfig.Zone
		}
		if zone == "" {
			zone = "asia-south2-c"
		}
	}
	return zone
}

// GetAKSLocation fetches the value of AKS Region;
// it first obtains the value from env var AKS_REGION, if the value is empty, it fetches the information from config file(cattle_config-import.yaml/cattle_config-provisioning.yaml)
// if none of the sources can provide a value, it returns the default value
func GetAKSLocation() string {
	region := os.Getenv("AKS_REGION")
	if region == "" {
		aksClusterConfig := new(management.AKSClusterConfigSpec)
		config.LoadConfig("aksClusterConfig", aksClusterConfig)
		region = aksClusterConfig.ResourceLocation
		if region == "" {
			region = "centralindia"
		}
	}
	return region
}

// GetEKSRegion fetches the value of EKS Region;
// it first obtains the value from env var EKS_REGION, if the value is empty, it fetches the information from config file(cattle_config-import.yaml/cattle_config-provisioning.yaml)
// if none of the sources can provide a value, it returns the default value
func GetEKSRegion() string {
	region := os.Getenv("EKS_REGION")
	if region == "" {
		eksClusterConfig := new(management.EKSClusterConfigSpec)
		config.LoadConfig("eksClusterConfig", eksClusterConfig)
		region = eksClusterConfig.Region
		if region == "" {
			region = "ap-south-1"
		}
	}
	return region
}

// GetGKEProjectID returns the value of GKE project by fetching the value of env var GKE_PROJECT_ID
func GetGKEProjectID() string {
	return os.Getenv("GKE_PROJECT_ID")
}

// GetCommonMetadataLabels returns a list of common metadata labels/tabs
func GetCommonMetadataLabels() map[string]string {
	testuser, err := user.Current()
	Expect(err).To(BeNil())

	specReport := ginkgo.CurrentSpecReport()
	// filename indicates the filename and line number of the test
	// we only use this information instead of the ginkgo.CurrentSpecReport().FullText() because of the 63 character limit
	var filename string
	// Because of the way Support Matrix suites are designed, filename is not loaded at first, so we need to ensure it is non-empty before sanitizing it
	//E.g. line51_k8s_chart_support_provisioning_test
	if specReport.FileName() != "" {
		// Sanitize the filename to fit the label requirements for all the hosted providers
		fileSplit := strings.Split(specReport.FileName(), "/") // abstract the filename
		filename = fileSplit[len(fileSplit)-1]
		filename = strings.TrimSuffix(filename, ".go") // `.` is not allowed
		filename = strings.ToLower(filename)           // string must be in lowercase
		filename = fmt.Sprintf("line%d_%s", specReport.LineNumber(), filename)
	}

	return map[string]string{
		"owner":          "hosted-providers-qa-ci-" + testuser.Username,
		"testfilenumber": filename,
	}
}

func ListOperatorChart() (operatorCharts []HelmChart) {
	output, err := kubectl.RunHelmBinaryWithOutput("list", "--namespace", CattleSystemNS, "-o", "json", "--filter", fmt.Sprintf("%s-operator", Provider))
	Expect(err).To(BeNil())
	ginkgo.GinkgoLogr.Info(output)
	err = json.Unmarshal([]byte(output), &operatorCharts)
	Expect(err).To(BeNil())
	for i := range operatorCharts {
		operatorCharts[i].DerivedVersion = strings.TrimPrefix(operatorCharts[i].Chart, fmt.Sprintf("%s-", operatorCharts[i].Name))
	}
	return
}

func VersionCompare(latestVersion, oldVersion string) int {
	latestVer, err := semver.ParseTolerant(latestVersion)
	Expect(err).To(BeNil())
	oldVer, err := semver.ParseTolerant(oldVersion)
	Expect(err).To(BeNil())
	return latestVer.Compare(oldVer)
}
