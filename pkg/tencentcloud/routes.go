package tencentcloud

import (
	"context"

	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"

	"k8s.io/apimachinery/pkg/types"
	cloudprovider "k8s.io/cloud-provider"
)

// ListRoutes lists all managed routes that belong to the specified clusterName
func (cloud *Cloud) ListRoutes(ctx context.Context, clusterName string) ([]*cloudprovider.Route, error) {
	//cloudRoutes, err := cloud.tke.DescribeClusterRoute(&tke.DescribeClusterRouteArgs{RouteTableName: cloud.txConfig.ClusterRouteTable})
	request := tke.NewDescribeClusterRouteTablesRequest()

	cloudRoutes, err := cloud.tke.DescribeClusterRouteTables(request)
	if err != nil {
		return []*cloudprovider.Route{}, err
	}

	routes := make([]*cloudprovider.Route, len(cloudRoutes.Data.RouteSet))

	for idx, route := range cloudRoutes.Data.RouteSet {
		routes[idx] = &cloudprovider.Route{Name: route.GatewayIp, TargetNode: types.NodeName(route.GatewayIp), DestinationCIDR: route.DestinationCidrBlock}
	}
	return routes, nil
}

// CreateRoute creates the described managed route
// route.Name will be ignored, although the cloud-provider may use nameHint
// to create a more user-meaningful name.
func (cloud *Cloud) CreateRoute(ctx context.Context, clusterName string, nameHint string, route *cloudprovider.Route) error {
	_, err := cloud.ccs.CreateClusterRoute(&tke.CreateClusterRouteArgs{
		RouteTableName:       cloud.txConfig.ClusterRouteTable,
		GatewayIp:            string(route.TargetNode),
		DestinationCidrBlock: route.DestinationCIDR,
	})

	return err
}

// DeleteRoute deletes the specified managed route
// Route should be as returned by ListRoutes
func (cloud *Cloud) DeleteRoute(ctx context.Context, clusterName string, route *cloudprovider.Route) error {
	_, err := cloud.ccs.DeleteClusterRoute(&tke.DeleteClusterRouteArgs{
		RouteTableName:       cloud.txConfig.ClusterRouteTable,
		GatewayIp:            string(route.TargetNode),
		DestinationCidrBlock: route.DestinationCIDR,
	})
	return err
}
