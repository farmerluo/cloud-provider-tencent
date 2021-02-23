package tencentcloud

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cloudErrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	cloudProvider "k8s.io/cloud-provider"
	"k8s.io/klog/v2"
)

// NodeAddresses returns the addresses of the specified instance.
// TODO(roberthbailey): This currently is only used in such a way that it
// returns the address of the calling instance. We should do a rename to
// make this clearer.
func (cloud *Cloud) NodeAddresses(ctx context.Context, name types.NodeName) ([]v1.NodeAddress, error) {

	node, err := cloud.getInstanceByInstancePrivateIp(ctx, string(name))
	if err != nil {
		klog.V(2).Infof("tencentcloud.NodeAddresses(\"%s\") message=[%v]", string(name), err)
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
	instance, err := cloud.getInstanceByProviderID(ctx, providerID)
	if err != nil {
		klog.V(2).Infof("tencentcloud.NodeAddressesByProviderID(\"%s\") message=[%v]", providerID, err)
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

// ExternalID returns the cloud provider ID of the node with the specified NodeName.
// Note that if the instance does not exist or is no longer running, we must return ("", cloudprovider.InstanceNotFound)
func (cloud *Cloud) ExternalID(ctx context.Context, nodeName types.NodeName) (string, error) {
	node, err := cloud.getInstanceByInstancePrivateIp(ctx, string(nodeName))
	if err != nil {
		klog.V(2).Infof("tencentcloud.ExternalID(\"%s\") message=[%v]", string(nodeName), err)
		return "", err
	}

	return *node.InstanceId, nil
}

// InstanceID returns the cloud provider ID of the node with the specified NodeName.
func (cloud *Cloud) InstanceID(ctx context.Context, nodeName types.NodeName) (string, error) {
	node, err := cloud.getInstanceByInstancePrivateIp(ctx, string(nodeName))
	if err != nil {
		klog.V(2).Infof("tencentcloud.InstanceID(\"%s\") message=[%v]", string(nodeName), err)
		return "", err
	}

	return fmt.Sprintf("/%s/%s", *node.Placement.Zone, *node.InstanceId), nil
}

// InstanceType returns the type of the specified instance.
func (cloud *Cloud) InstanceType(ctx context.Context, name types.NodeName) (string, error) {
	node, err := cloud.getInstanceByInstancePrivateIp(ctx, string(name))
	if err != nil {
		klog.V(2).Infof("tencentcloud.InstanceType(\"%s\") message=[%v]", string(name), err)
		return "", err
	}

	return *node.InstanceType, nil
}

// InstanceTypeByProviderID returns the type of the specified instance.
func (cloud *Cloud) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	node, err := cloud.getInstanceByProviderID(ctx, providerID)
	if err != nil {
		klog.V(2).Infof("tencentcloud.InstanceTypeByProviderID(\"%s\") message=[%v]", providerID, err)
		return "", err
	}

	return *node.InstanceType, nil
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
	_, err := cloud.getInstanceByProviderID(ctx, providerID)
	if err == cloudProvider.InstanceNotFound {
		klog.V(2).Infof("tencentcloud.InstanceExistsByProviderID(\"%s\") message=[%v]", providerID, err)
		return false, err
	}
	return true, err
}

// InstanceShutdownByProviderID returns true if the instance is shutdown in cloudprovider
func (cloud *Cloud) InstanceShutdownByProviderID(ctx context.Context, providerID string) (bool, error) {
	instance, err := cloud.getInstanceByProviderID(ctx, providerID)
	if err != nil {
		klog.V(2).Infof("tencentcloud.InstanceShutdownByProviderID(\"%s\") message=[%v]", providerID, err)
		return false, err
	}
	if *instance.InstanceState != "RUNNING" {
		return true, err
	}
	return false, err
}

func (cloud *Cloud) getInstanceByInstancePrivateIp(ctx context.Context, privateIp string) (*cvm.Instance, error) {
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
	return nil, cloudProvider.InstanceNotFound
}

func (cloud *Cloud) getInstanceByInstanceID(ctx context.Context, instanceID string) (*cvm.Instance, error) {
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
	return nil, cloudProvider.InstanceNotFound

}

// getInstanceIdByProviderID returns the addresses of the specified instance.
func (cloud *Cloud) getInstanceByProviderID(ctx context.Context, providerID string) (*cvm.Instance, error) {
	id := strings.TrimPrefix(providerID, fmt.Sprintf("%s://", providerName))
	parts := strings.Split(id, "/")
	if len(parts) == 3 {
		instance, err := cloud.getInstanceByInstanceID(ctx, parts[2])
		if err != nil {
			return nil, err
		}
		return instance, nil
	}
	return nil, errors.New(fmt.Sprintf("invalid format for providerId %s", providerID))
}
