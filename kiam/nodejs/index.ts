import * as pulumi from "@pulumi/pulumi";
import * as aws from "@pulumi/aws";
import * as k8s from "@pulumi/kubernetes";

const kiamNodeRole = new aws.iam.Role("kiamNodeRole", {
    assumeRolePolicy: JSON.stringify({
        Version: "2012-10-17",
        Statement: [{
            Action: "sts:AssumeRole",
            Principal: {
                Service: "ec2.amazonaws.com",
            },
            Effect: "Allow",
        }],
    })
});

const kiamNodePolicy = new aws.iam.RolePolicy("kiamNodePolicy", {
    role: kiamNodeRole.name,
    policy: kiamNodeRole.arn.apply(arn => JSON.stringify({
        Version: "2012-10-17",
        Statement: [{
            Effect: "Allow",
            Action: "sts:AssumeRole",
            Resource: arn,
        }],
    })),
}, {parent: kiamNodeRole});

const kiamNodeProfile = new aws.iam.InstanceProfile("kiamNodeProfile", {
    role: kiamNodeRole.name
}, {parent: kiamNodeRole});

const kiamServerRole = new aws.iam.Role("kiamServerRole", {
    assumeRolePolicy: kiamNodeRole.arn.apply(arn => JSON.stringify({
        Version: "2012-10-17",
        Statement: [{
            Sid: "",
            Effect: "Allow",
            Principal: {
                AWS: arn,
            },
            Action: "sts:AssumeRole",
        }]
    }))
});

const kiamServerPolicy = new aws.iam.Policy("kiamServerPolicy", {
    policy: JSON.stringify({
        Version: "2012-10-17",
        Statement: [{
            Effect: "Allow",
            Action: "sts:AssumeRole",
            Resource: "*",
        }]
    })
}, {parent: kiamServerRole});

const kiamServerPolicyAttachment = new aws.iam.PolicyAttachment("kiamServerPolicyAttachment", {
    roles: [kiamServerRole.name],
    policyArn: kiamServerPolicy.arn,
});

const namespace = new k8s.core.v1.Namespace("kiam", {
    metadata: {
        name: 'kiam',
        annotations: {
            'iam.amazonaws.com/permitted': ".*"
        }
    }
});

const kiam = new k8s.helm.v3.Chart('kiam', {
    namespace: namespace.metadata.name,
    chart: "kiam",
    fetchOpts: {repo: "https://uswitch.github.io/kiam-helm-charts/charts/"},
    values: {
        agent: {
            host: {
                iptables: true,
                interface: "!eth0"
            }
        },
        server: {
            assumeRoleArn: kiamServerRole.arn.apply(arn => arn),
            useHostNetwork: true,
        }
    }
}, {parent: namespace})
