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

// Annotations returns the Installation annotation catalog.
func Annotations() []v1alpha1.Annotation {
	return []v1alpha1.Annotation{
		SignozBinaryPath,
		IngesterBinaryPath,
		MetaStorePostgresBinaryPath,
		TelemetryStoreClickHouseBinaryPath,
		TelemetryKeeperClickHouseKeeperBinaryPath,
		TelemetryKeeperZookeeperBinaryPath,
		MCPBinaryPath,
	}
}
