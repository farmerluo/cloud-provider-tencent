package tencentcloud

import (
	"context"
	"fmt"

	"errors"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cloudErrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

	"strings"

	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cloudProvider "k8s.io/cloud-provider"
)

// NodeAddresses returns the addresses of the specified instance.
// TODO(roberthbailey): This currently is only used in such a way that it
// returns the address of the calling instance. We should do a rename to
// make this clearer.
func (cloud *Cloud) NodeAddresses(ctx context.Context, name types.NodeName) ([]v1.NodeAddress, error) {

	node, err := cloud.getInstanceByInstancePrivateIp(string(name))
	if err != nil {
		return []v1.NodeAddress{}, err
	}
	addresses := make([]v1.NodeAddress, len(node.PrivateIpAddresses)+len(node.PublicIpAddresses))
	for idx, ip := range node.PrivateIpAddresses {
		addresses[idx] = v1.NodeAddress{Type: v1.NodeInternalIP, Address: *ip}
	}
	for idx, ip := range node.PublicIpAddresses {
		addresses[len(node.PrivateIpAddresses)+idx] = v1.NodeAddress{Type: v1.NodeExternalIP, Address: *ip}
	}
	return addresses, nil
}

// NodeAddressesByProviderID returns the addresses of the specified instance.
// The instance is specified using the providerID of the node. The
// ProviderID is a unique identifier of the node. This will not be called
// from the node whose nodeaddresses are being queried. i.e. local metadata
// services cannot be used in this method to obtain nodeaddresses
func (cloud *Cloud) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {
	id := strings.TrimPrefix(providerID, fmt.Sprintf("%s://", providerName))
	parts := strings.Split(id, "/")
	if len(parts) == 3 {
		instance, err := cloud.getInstanceByInstanceID(parts[2])
		if err != nil {
			return []v1.NodeAddress{}, err
		}
		addresses := make([]v1.NodeAddress, len(instance.PrivateIpAddresses)+len(instance.PublicIpAddresses))
		for idx, ip := range instance.PrivateIpAddresses {
			addresses[idx] = v1.NodeAddress{Type: v1.NodeInternalIP, Address: *ip}
		}
		for idx, ip := range instance.PublicIpAddresses {
			addresses[len(instance.PrivateIpAddresses)+idx] = v1.NodeAddress{Type: v1.NodeExternalIP, Address: *ip}
		}
		return addresses, nil
	}
	return []v1.NodeAddress{}, errors.New(fmt.Sprintf("invalid format for providerId %s", providerID))
}

// ExternalID returns the cloud provider ID of the node with the specified NodeName.
// Note that if the instance does not exist or is no longer running, we must return ("", cloudprovider.InstanceNotFound)
func (cloud *Cloud) ExternalID(ctx context.Context, nodeName types.NodeName) (string, error) {
	node, err := cloud.getInstanceByInstancePrivateIp(string(nodeName))
	if err != nil {
		return "", err
	}

	return *node.InstanceId, nil
}

// InstanceID returns the cloud provider ID of the node with the specified NodeName.
func (cloud *Cloud) InstanceID(ctx context.Context, nodeName types.NodeName) (string, error) {
	node, err := cloud.getInstanceByInstancePrivateIp(string(nodeName))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("/%s/%s", *node.Placement.Zone, *node.InstanceId), nil
}

// InstanceType returns the type of the specified instance.
func (cloud *Cloud) InstanceType(ctx context.Context, name types.NodeName) (string, error) {
	return providerName, nil
}

// InstanceTypeByProviderID returns the type of the specified instance.
func (cloud *Cloud) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	return providerName, nil
}

// AddSSHKeyToAllInstances adds an SSH public key as a legal identity for all instances
// expected format for the key is standard ssh-keygen format: <protocol> <blob>
func (cloud *Cloud) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudProvider.NotImplemented
}

// CurrentNodeName returns the name of the node we are currently running on
// On most clouds (e.g. GCE) this is the hostname, so we provide the hostname
func (cloud *Cloud) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {
	return types.NodeName(""), cloudProvider.NotImplemented
}

// InstanceExistsByProviderID returns true if the instance for the given provider id still is running.
// If false is returned with no error, the instance will be immediately deleted by the cloud controller manager.
func (cloud *Cloud) InstanceExistsByProviderID(ctx context.Context, providerID string) (bool, error) {
	return true, nil
}

// InstanceExistsByProviderID returns true if the instance for the given provider id still is running.
// If false is returned with no error, the instance will be immediately deleted by the cloud controller manager.
func (cloud *Cloud) InstanceShutdownByProviderID(ctx context.Context, providerID string) (bool, error) {
	return true, nil
}

func (cloud *Cloud) getInstanceByInstancePrivateIp(privateIp string) (*cvm.Instance, error) {
	request := cvm.NewDescribeInstancesRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Values: common.StringPtrs([]string{privateIp}),
			Name:   common.StringPtr("private-ip-address"),
		},
	}

	response, err := cloud.cvm.DescribeInstances(request)
	if _, ok := err.(*cloudErrors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	//fmt.Printf("%s", response.ToJsonString())

	for _, instance := range response.Response.InstanceSet {
		if *instance.VirtualPrivateCloud.VpcId != cloud.txConfig.VpcId {
			continue
		}
		for _, ip := range instance.PrivateIpAddresses {
			if *ip == privateIp {
				return instance, nil
			}
		}
	}
	return nil, CloudInstanceNotFound
}

func (cloud *Cloud) getInstanceByInstanceID(instanceID string) (*cvm.Instance, error) {
	request := cvm.NewDescribeInstancesRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Values: common.StringPtrs([]string{instanceID}),
			Name:   common.StringPtr("instance-id"),
		},
	}

	response, err := cloud.cvm.DescribeInstances(request)
	if _, ok := err.(*cloudErrors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	//fmt.Printf("%s", response.ToJsonString())

	for _, instance := range response.Response.InstanceSet {
		if *instance.VirtualPrivateCloud.VpcId != cloud.txConfig.VpcId {
			continue
		}
		if instanceID == *instance.InstanceId {
			return instance, nil
		}
	}
	return nil, CloudInstanceNotFound

}

// getInstanceIdByProviderID returns the addresses of the specified instance.
func (cloud *Cloud) getInstanceByProviderID(providerID string) (*cvm.Instance, error) {
	id := strings.TrimPrefix(providerID, fmt.Sprintf("%s://", providerName))
	parts := strings.Split(id, "/")
	if len(parts) == 3 {
		instance, err := cloud.getInstanceByInstanceID(parts[2])
		if err != nil {
			return nil, err
		}
		return instance, nil
	}
	return nil, errors.New(fmt.Sprintf("invalid format for providerId %s", providerID))
}
