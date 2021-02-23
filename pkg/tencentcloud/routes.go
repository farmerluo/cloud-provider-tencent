package tencentcloud

import (
	"context"
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"k8s.io/klog/v2"

	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"

	"k8s.io/apimachinery/pkg/types"
	cloudProvider "k8s.io/cloud-provider"
)

// ListRoutes lists all managed routes that belong to the specified clusterName
func (cloud *Cloud) ListRoutes(ctx context.Context, clusterName string) ([]*cloudProvider.Route, error) {
	klog.V(5).Infof("tencentcloud: ListRoutes(\"%s\")\n", clusterName)
	//cloudRoutes, err := cloud.tke.DescribeClusterRoute(&tke.DescribeClusterRouteArgs{RouteTableName: cloud.txConfig.ClusterRouteTable})
	request := tke.NewDescribeClusterRoutesRequest()
	request.RouteTableName = common.StringPtr(cloud.txConfig.ClusterRouteTable)

	cloudRoutes, err := cloud.tke.DescribeClusterRoutes(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
		return []*cloudProvider.Route{}, err
	}
	//fmt.Printf("%s", cloudRoutes.ToJsonString())
	if err != nil {
		return []*cloudProvider.Route{}, err
	}

	routes := make([]*cloudProvider.Route, len(cloudRoutes.Response.RouteSet))
	for idx, route := range cloudRoutes.Response.RouteSet {
		routes[idx] = &cloudProvider.Route{Name: *route.GatewayIp, TargetNode: types.NodeName(*route.GatewayIp), DestinationCIDR: *route.DestinationCidrBlock}
	}
	return routes, nil
}

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.
func (cloud *Cloud) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudProvider.Route) error {
	klog.V(2).Infof("tencentcloud: CreateRoute(\"%s, %s, %v\")\n", clusterName, nameHint, route)

	request := tke.NewCreateClusterRouteRequest()
	request.RouteTableName = common.StringPtr(cloud.txConfig.ClusterRouteTable)
	request.GatewayIp = common.StringPtr(string(route.TargetNode))
	request.DestinationCidrBlock = common.StringPtr(route.DestinationCIDR)

	_, err := cloud.tke.CreateClusterRoute(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
	}

	return err
}

// DeleteRoute deletes the specified managed route
// Route should be as returned by ListRoutes
func (cloud *Cloud) DeleteRoute(ctx context.Context, clusterName string, route *cloudProvider.Route) error {
	klog.V(2).Infof("tencentcloud: DeleteRoute(\"%s, %v\")\n", clusterName, route)

	request := tke.NewDeleteClusterRouteRequest()
	request.RouteTableName = common.StringPtr(cloud.txConfig.ClusterRouteTable)
	request.GatewayIp = common.StringPtr(string(route.TargetNode))
	request.DestinationCidrBlock = common.StringPtr(route.DestinationCIDR)

	_, err := cloud.tke.DeleteClusterRoute(request)
	if _, ok := err.(*errors.TencentCloudSDKError); ok {
		fmt.Printf("An API error has returned: %s", err)
	}

	return err
}
