package main

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v3/go/aws/iam"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"

	"github.com/pulumi/pulumi-kubernetes/sdk/v2/go/kubernetes/helm/v3"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		nodePolicyRawJSON, err := json.Marshal(map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				map[string]interface{}{
					"Action": "sts:AssumeRole",
					"Principal": map[string]interface{}{
						"Service": "ec2.amazonaws.com",
					},
					"Effect": "Allow",
				},
			},
		})

		if err != nil {
			return err
		}

		nodePolicyJSON := string(nodePolicyRawJSON)
		kiamNodeRole, err := iam.NewRole(ctx, "kiamNodeRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(nodePolicyJSON),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicy(ctx, "kiamNodePolicy", &iam.RolePolicyArgs{
			Role: kiamNodeRole.Name,
			Policy: kiamNodeRole.Arn.ApplyT(func(arn string) (pulumi.String, error) {
				var _zero pulumi.String
				nodeRoleJSONRaw, err := json.Marshal(map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []map[string]interface{}{
						map[string]interface{}{
							"Effect":   "Allow",
							"Action":   "sts:AssumeRole",
							"Resource": arn,
						},
					},
				})
				if err != nil {
					return _zero, err
				}
				nodeRoleJSON := string(nodeRoleJSONRaw)
				return pulumi.String(nodeRoleJSON), nil
			}),
		}, pulumi.Parent(kiamNodeRole))

		if err != nil {
			return err
		}


		_, err = iam.NewInstanceProfile(ctx, "kiamNodeProfile", &iam.InstanceProfileArgs{
			Role: kiamNodeRole.Name,
		}, pulumi.Parent(kiamNodeRole))
		if err != nil {
			return err
		}
		kiamServerRole, err := iam.NewRole(ctx, "kiamServerRole", &iam.RoleArgs{
			AssumeRolePolicy: kiamNodeRole.Arn.ApplyT(func(arn string) (pulumi.String, error) {
				var _zero pulumi.String
				serverRoleJSONRaw, err := json.Marshal(map[string]interface{}{
					"Version": "2012-10-17",
					"Statement": []map[string]interface{}{
						map[string]interface{}{
							"Sid":    "",
							"Effect": "Allow",
							"Principal": map[string]interface{}{
								"AWS": arn,
							},
							"Action": "sts:AssumeRole",
						},
					},
				})
				if err != nil {
					return _zero, err
				}
				serverRoleJSON := string(serverRoleJSONRaw)
				return pulumi.String(serverRoleJSON), nil
			}),
		})
		if err != nil {
			return err
		}

		serverPolicyJSONRaw, err := json.Marshal(map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				map[string]interface{}{
					"Effect":   "Allow",
					"Action":   "sts:AssumeRole",
					"Resource": "*",
				},
			},
		})
		if err != nil {
			return err
		}

		serverPolicyJSON := string(serverPolicyJSONRaw)
		kiamServerPolicy, err := iam.NewPolicy(ctx, "kiamServerPolicy", &iam.PolicyArgs{
			Policy: pulumi.String(serverPolicyJSON),
		}, pulumi.Parent(kiamServerRole))
		if err != nil {
			return err
		}

		_, err = iam.NewPolicyAttachment(ctx, "kiamServerPolicyAttachment", &iam.PolicyAttachmentArgs{
			Roles: pulumi.Array{
				kiamServerRole.Name,
			},
			PolicyArn: kiamServerPolicy.Arn,
		}, pulumi.Parent(kiamServerRole))
		if err != nil {
			return err
		}

		namespace, err := corev1.NewNamespace(ctx, "kiam", &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String("kiam"),
			},
		})

		_, err = helm.NewChart(ctx, "kiam", helm.ChartArgs{
			Chart: pulumi.String("kiam"),
			FetchArgs: &helm.FetchArgs{
				Repo: pulumi.String("https://uswitch.github.io/kiam-helm-charts/charts/"),
			},
			Values: pulumi.Map{
				"agent": pulumi.Map{
					"host": pulumi.Map{
						"iptables": pulumi.String("true"),
						"interface": pulumi.String("!eth0"),
					},
				},
				"server": pulumi.Map{
					"useHostNetwork": pulumi.String("true"),
					"assumeRoleArn": kiamServerRole.Arn.ApplyT(func(arn string) (pulumi.String, error){
						var empty pulumi.String
						if err != nil {
							return empty, err
						}
						return pulumi.String(arn), nil
					}),
				},
			},
			Namespace: namespace.Metadata.Name().Elem(),
		}, pulumi.Parent(namespace))

		return nil
	})
}
