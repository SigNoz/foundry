package installation

import "github.com/signoz/foundry/api/v1alpha1"

// Binary path annotations for the systemd/binary deployment of the Installation
// Kind. Each resolves the absolute path to a component's executable; omitting it
// falls back to the default. This is the single source for these keys and
// defaults, consumed by the enricher, the unit templates, and the binary check.
var (
	SignozBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/signoz-binary-path",
		Default:     "/opt/signoz/bin/signoz",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the SigNoz server binary.",
	}
	IngesterBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ingester-binary-path",
		Default:     "/opt/ingester/bin/signoz-otel-collector",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the SigNoz OTel Collector (ingester) binary.",
	}
	MetaStorePostgresBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/metastore-postgres-binary-path",
		Default:     "/usr/bin/postgres",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the PostgreSQL server binary; its bindir also holds initdb and pg_ctl.",
	}
	TelemetryStoreClickHouseBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/telemetrystore-clickhouse-binary-path",
		Default:     "/usr/bin/clickhouse",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the ClickHouse binary, run as `clickhouse server`.",
	}
	TelemetryKeeperClickHouseKeeperBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/telemetrykeeper-clickhousekeeper-binary-path",
		Default:     "/usr/bin/clickhouse",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the ClickHouse binary, run as `clickhouse keeper`.",
	}
	TelemetryKeeperZookeeperBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/telemetrykeeper-zookeeper-binary-path",
		Default:     "/opt/zookeeper/bin/zkServer.sh",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the Apache ZooKeeper zkServer.sh script; requires a Java runtime on the host.",
	}
	MCPBinaryPath = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/mcp-binary-path",
		Default:     "/opt/mcp/bin/signoz-mcp-server",
		Mode:        v1alpha1.ModeSystemd,
		Description: "Absolute path to the SigNoz MCP server binary.",
	}
)

// Cluster annotations for the ECS/EC2 deployment of the Installation Kind. The
// casting places tasks onto a cluster it does not provision, so each of these
// names an existing AWS object it must be handed. They have no defaults: an
// absent value is a missing cluster, not a fallback.
var (
	ECSRegion = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-region",
		Mode:        v1alpha1.ModeEC2,
		Description: "AWS region holding the cluster.",
	}
	ECSClusterARN = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-cluster-arn",
		Mode:        v1alpha1.ModeEC2,
		Description: "ARN of the ECS cluster to deploy services into.",
	}
	ECSSubnetIDs = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-subnet-ids",
		Mode:        v1alpha1.ModeEC2,
		Description: "Comma-separated subnet IDs for task networking (awsvpc).",
	}
	ECSSecurityGroupIDs = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-security-group-ids",
		Mode:        v1alpha1.ModeEC2,
		Description: "Comma-separated security group IDs for task networking (awsvpc); must permit intra-cluster traffic.",
	}
	ECSVPCID = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-vpc-id",
		Mode:        v1alpha1.ModeEC2,
		Description: "VPC ID the Cloud Map private DNS namespace is created in.",
	}
	ECSTaskRoleARN = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-task-role-arn",
		Mode:        v1alpha1.ModeEC2,
		Description: "IAM role ARN assumed by the tasks; needs read access to AWS AppConfig.",
	}
	ECSTaskExecutionRoleARN = v1alpha1.Annotation{
		Key:         "foundry.signoz.io/ecs-task-execution-role-arn",
		Mode:        v1alpha1.ModeEC2,
		Description: "IAM role ARN the ECS agent assumes to pull images and start tasks.",
	}
)

// Annotations returns the Installation annotation catalog.
func Annotations() []v1alpha1.Annotation {
	return []v1alpha1.Annotation{
		SignozBinaryPath,
		IngesterBinaryPath,
		MetaStorePostgresBinaryPath,
		TelemetryStoreClickHouseBinaryPath,
		TelemetryKeeperClickHouseKeeperBinaryPath,
		ECSRegion,
		ECSClusterARN,
		ECSSubnetIDs,
		ECSSecurityGroupIDs,
		ECSVPCID,
		ECSTaskRoleARN,
		ECSTaskExecutionRoleARN,
		TelemetryKeeperZookeeperBinaryPath,
		MCPBinaryPath,
	}
}
