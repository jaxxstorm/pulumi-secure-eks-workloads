using System.Collections.Generic;
using System.Text.Json;
using Pulumi;
using Aws = Pulumi.Aws;
using Pulumi.Kubernetes.Core.V1;
using Pulumi.Kubernetes.Helm;
using Pulumi.Kubernetes.Helm.V3;
using Pulumi.Kubernetes.Types.Outputs.Meta.V1;
using Pulumi.Kubernetes.Types.Inputs.Core.V1;
using Pulumi.Kubernetes.Types.Inputs.Meta.V1;

class MyStack : Stack
{
    public MyStack()
    {
        var kiamNodeRole = new Aws.Iam.Role("kiamNodeRole", new Aws.Iam.RoleArgs
        {
            AssumeRolePolicy = JsonSerializer.Serialize(new Dictionary<string, object?>
            {
                { "Version", "2012-10-17" },
                { "Statement", new[]
                    {
                        new Dictionary<string, object?>
                        {
                            { "Action", "sts:AssumeRole" },
                            { "Principal", new Dictionary<string, object?>
                            {
                                { "Service", "ec2.amazonaws.com" },
                            } },
                            { "Effect", "Allow" },
                        },
                    }
                 },
            }),
        });
        var kiamNodePolicy = new Aws.Iam.RolePolicy("kiamNodePolicy", new Aws.Iam.RolePolicyArgs
        {
            Role = kiamNodeRole.Name,
            Policy = kiamNodeRole.Arn.Apply(arn => JsonSerializer.Serialize(new Dictionary<string, object?>
            {
                { "Version", "2012-10-17" },
                { "Statement", new[]
                    {
                        new Dictionary<string, object?>
                        {
                            { "Effect", "Allow" },
                            { "Action", "sts:AssumeRole" },
                            { "Resource", arn },
                        },
                    }
                 },
            })),
        }, new CustomResourceOptions
        {
            Parent = kiamNodeRole,
        });
        var kiamNodeProfile = new Aws.Iam.InstanceProfile("kiamNodeProfile", new Aws.Iam.InstanceProfileArgs
        {
            Role = kiamNodeRole.Name,
        }, new CustomResourceOptions
        {
            Parent = kiamNodeRole,
        });
        var kiamServerRole = new Aws.Iam.Role("kiamServerRole", new Aws.Iam.RoleArgs
        {
            AssumeRolePolicy = kiamNodeRole.Arn.Apply(arn => JsonSerializer.Serialize(new Dictionary<string, object?>
            {
                { "Version", "2012-10-17" },
                { "Statement", new[]
                    {
                        new Dictionary<string, object?>
                        {
                            { "Sid", "" },
                            { "Effect", "Allow" },
                            { "Principal", new Dictionary<string, object?>
                            {
                                { "AWS", arn },
                            } },
                            { "Action", "sts:AssumeRole" },
                        },
                    }
                 },
            })),
        });
        var kiamServerPolicy = new Aws.Iam.Policy("kiamServerPolicy", new Aws.Iam.PolicyArgs
        {
            PolicyDocument = JsonSerializer.Serialize(new Dictionary<string, object?>
            {
                { "Version", "2012-10-17" },
                { "Statement", new[]
                    {
                        new Dictionary<string, object?>
                        {
                            { "Effect", "Allow" },
                            { "Action", "sts:AssumeRole" },
                            { "Resource", "*" },
                        },
                    }
                 },
            }),
        }, new CustomResourceOptions
        {
            Parent = kiamServerRole,
        });
        var kiamServerPolicyAttachment = new Aws.Iam.PolicyAttachment("kiamServerPolicyAttachment", new Aws.Iam.PolicyAttachmentArgs
        {
            Roles = 
            {
                kiamServerRole.Name,
            },
            PolicyArn = kiamServerPolicy.Arn,
        }, new CustomResourceOptions
        {
            Parent = kiamServerRole,
        });
        
        var ns = new Pulumi.Kubernetes.Core.V1.Namespace("kiam", new NamespaceArgs
        {
            Metadata = new ObjectMetaArgs
            {
                Name = "kiam"
            }
        });
        
        var kiam = new Pulumi.Kubernetes.Helm.V3.Chart("kiam", new ChartArgs
        {
            Chart = "kiam",
            Namespace = ns.Metadata.Apply(n => n.Name),
            FetchOptions = new ChartFetchArgs
            {
                Repo = "https://uswitch.github.io/kiam-helm-charts/charts/"
            },
            Values = new Dictionary<string, object>
            {
                ["agent"] = new Dictionary<string, object>
                {
                    ["host"] = new Dictionary<string, object>
                    {
                        ["iptables"] = "true",
                        ["interface"] = "!eth0",
                    }
                }
                ["server"] = new Dictionary<string, object>
                {
                    ["useHostNetwork"] = "true",
                    ["assumeRoleArn"] = kiamServerRole.Arn.Apply(arn => arn)
                }
            }
        });


    }
}
