package tencentcloud

import (
	"encoding/json"
	"io"
	"io/ioutil"
	cloudprovider "k8s.io/cloud-provider"
	"os"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	ccs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"

	"k8s.io/client-go/kubernetes"
)

const (
	providerName = "tencentcloud"
)

type TxCloudConfig struct {
	Region string `json:"region"`
	VpcId  string `json:"vpc_id"`

	SecretId  string `json:"secret_id"`
	SecretKey string `json:"secret_key"`

	ClusterRouteTable string `json:"cluster_route_table"`
}

type Cloud struct {
	config TxCloudConfig

	kubeClient kubernetes.Interface

	cvm   *cvm.Client
	cvmV3 *cvm.Client
	ccs   *ccs.Client
	clb   *clb.Client
}


func NewCloud(config io.Reader) (*Cloud, error) {
	var c TxCloudConfig
	if config != nil {
		cfg, err := ioutil.ReadAll(config)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(cfg, &c); err != nil {
			return nil, err
		}
	}

	if c.Region == "" {
		c.Region = os.Getenv("TENCENTCLOUD_CLOUD_CONTROLLER_MANAGER_REGION")
	}
	if c.VpcId == "" {
		c.VpcId = os.Getenv("TENCENTCLOUD_CLOUD_CONTROLLER_MANAGER_VPC_ID")
	}
	if c.SecretId == "" {
		c.SecretId = os.Getenv("TENCENTCLOUD_CLOUD_CONTROLLER_MANAGER_SECRET_ID")
	}
	if c.SecretKey == "" {
		c.SecretKey = os.Getenv("TENCENTCLOUD_CLOUD_CONTROLLER_MANAGER_SECRET_KEY")
	}

	if c.ClusterRouteTable == "" {
		c.ClusterRouteTable = os.Getenv("TENCENTCLOUD_CLOUD_CONTROLLER_MANAGER_CLUSTER_ROUTE_TABLE")
	}

	return &Cloud{config: c}, nil
}

func init() {
	cloudprovider.RegisterCloudProvider(providerName, func(config io.Reader) (cloudprovider.Interface, error) {
		return NewCloud(config)
	})
}

// Initialize provides the cloud with a kubernetes client builder and may spawn goroutines
// to perform housekeeping activities within the cloud provider.
func (cloud *Cloud) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	cloud.kubeClient = clientBuilder.ClientOrDie("tencentcloud-cloud-provider")
	credential := common.NewCredential(
		//os.Getenv("TENCENTCLOUD_SECRET_ID"),
		//os.Getenv("TENCENTCLOUD_SECRET_KEY"),
		cloud.config.SecretId,
		cloud.config.SecretKey,
	)
	// 非必要步骤
	// 实例化一个客户端配置对象，可以指定超时时间等配置
	cpf := profile.NewClientProfile()
	// SDK有默认的超时时间，非必要请不要进行调整。
	// 如有需要请在代码中查阅以获取最新的默认值。
	cpf.HttpProfile.ReqTimeout = 10
	cvmClient, err := cvm.NewClient(credential, cloud.config.Region, cpf)
	if err != nil {
		panic(err)
	}
	cloud.cvm = cvmClient
	cvmV3Client, err := cvm.NewClient(credential, cloud.config.Region, cpf)
	if err != nil {
		panic(err)
	}
	cloud.cvmV3 = cvmV3Client
	ccsClient, err := ccs.NewClient(credential, cloud.config.Region, cpf)
	if err != nil {
		panic(err)
	}
	cloud.ccs = ccsClient
	clbClient, err := clb.NewClient(credential, cloud.config.Region, cpf)
	if err != nil {
		panic(err)
	}
	cloud.clb = clbClient
	return
}

// LoadBalancer returns a balancer interface. Also returns true if the interface is supported, false otherwise.
func (cloud *Cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return nil, false
}

// Instances returns an instances interface. Also returns true if the interface is supported, false otherwise.
func (cloud *Cloud) Instances() (cloudprovider.Instances, bool) {
	return cloud, true
}

// Zones returns a zones interface. Also returns true if the interface is supported, false otherwise.
func (cloud *Cloud) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

// Clusters returns a clusters interface.  Also returns true if the interface is supported, false otherwise.
func (cloud *Cloud) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes returns a routes interface along with whether the interface is supported.
func (cloud *Cloud) Routes() (cloudprovider.Routes, bool) {
	return cloud, true
}

// ProviderName returns the cloud provider ID.
func (cloud *Cloud) ProviderName() string {
	return providerName
}

// HasClusterID returns true if a ClusterID is required and set
func (cloud *Cloud) HasClusterID() bool {
	return false
}
