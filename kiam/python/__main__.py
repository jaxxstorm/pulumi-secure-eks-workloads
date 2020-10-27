import pulumi
import json
import pulumi_aws as aws
import pulumi_kubernetes as k8s
import pulumi_kubernetes.helm.v3 as helm

kiam_node_role = aws.iam.Role(
    "kiamNodeRole",
    assume_role_policy=json.dumps(
        {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Action": "sts:AssumeRole",
                    "Principal": {
                        "Service": "ec2.amazonaws.com",
                    },
                    "Effect": "Allow",
                }
            ],
        }
    ),
)

kiam_node_policy = aws.iam.RolePolicy(
    "kiamNodePolicy",
    role=kiam_node_role.name,
    policy=kiam_node_role.arn.apply(
        lambda arn: json.dumps(
            {
                "Version": "2012-10-17",
                "Statement": [
                    {
                        "Effect": "Allow",
                        "Action": "sts:AssumeRole",
                        "Resource": arn,
                    }
                ],
            }
        )
    ),
    opts=pulumi.ResourceOptions(parent=kiam_node_role),
)

kiam_node_profile = aws.iam.InstanceProfile(
    "kiamNodeProfile",
    role=kiam_node_role.name,
    opts=pulumi.ResourceOptions(parent=kiam_node_role),
)
kiam_server_role = aws.iam.Role(
    "kiamServerRole",
    assume_role_policy=kiam_node_role.arn.apply(
        lambda arn: json.dumps(
            {
                "Version": "2012-10-17",
                "Statement": [
                    {
                        "Sid": "",
                        "Effect": "Allow",
                        "Principal": {
                            "AWS": arn,
                        },
                        "Action": "sts:AssumeRole",
                    }
                ],
            }
        )
    ),
)
kiam_server_policy = aws.iam.Policy(
    "kiamServerPolicy",
    policy=json.dumps(
        {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": "sts:AssumeRole",
                    "Resource": "*",
                }
            ],
        }
    ),
    opts=pulumi.ResourceOptions(parent=kiam_server_role),
)
kiam_server_policy_attachment = aws.iam.PolicyAttachment(
    "kiamServerPolicyAttachment",
    roles=[kiam_server_role.name],
    policy_arn=kiam_server_policy.arn,
)

namespace = k8s.core.v1.Namespace(
    "kiam",
    metadata={"name": "kiam", "annotations": {"iam.amazonaws.com/permitted": ".*"}},
)

kiam = helm.Chart("kiam", helm.ChartOpts(
    namespace=namespace.metadata.name,
    chart="kiam",
    fetch_opts=helm.FetchOpts(
        repo="https://uswitch.github.io/kiam-helm-charts/charts",
    ),
    values={
        "agent": {
            "host": {
                "iptables": "true",
                "interface": "!eth0",
            }
        },
        "server": {
            'useHostNetwork': 'true',
            'assumeRoleArn': kiam_server_role.arn.apply(lambda arn: arn)
        }
    }
))

# kiam = helm.Chart(
#     "kiam",
#     helm.ChartOpts(
#         namespace=namespace.metadata.name,
#         chart="kiam",
#         fetch_opts=helm.FetchOpts(
#             "repo"="https://uswitch.github.io/kiam-helm-charts/charts/",
#         ),
#         values={
#             "agent": {
#                 "host": {
#                     "iptables": "true",
#                     "interface": "!eth0",
#                 }
#             },
#             "server": {
#                 'useHostNetwork': 'true',
#                 'assumeRoleArn': kiam_server_role.arn.apply(lambda arn: arn)
#             },
#         },
#     ),
# )
