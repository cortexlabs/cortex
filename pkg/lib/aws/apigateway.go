package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/apigatewayv2"
)

func (c *Client) getVpcLinkID(clusterName string) (string, error) {

	vpcLinks, err := c.APIGatewayv2().GetVpcLinks(&apigatewayv2.GetVpcLinksInput{})
	if err != nil {
		return "", err
	}
	for _, vpcLink := range vpcLinks.Items {
		for tag, value := range vpcLink.Tags {
			fmt.Println(tag, value)
			if tag == "cortex.dev/cluster-name" && *value == clusterName {
				return *vpcLink.VpcLinkId, nil
			}

		}
	}
	return "",errors.

}
