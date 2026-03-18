package ecstaskdefcasting

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/signoz/foundry/api/v1alpha1"
	rootcasting "github.com/signoz/foundry/internal/casting"
)

const (
	annotationCluster          = "foundry.signoz.io/ecs-cluster"
	annotationRegion           = "foundry.signoz.io/ecs-region"
	annotationSubnetIDs        = "foundry.signoz.io/ecs-subnet-ids"
	annotationSecurityGroupIDs = "foundry.signoz.io/ecs-security-group-ids"
	annotationCapacityProvider = "foundry.signoz.io/ecs-capacity-provider"
	annotationConfigBucket     = "foundry.signoz.io/ecs-config-bucket"
	annotationServiceConnectNS = "foundry.signoz.io/ecs-service-connect-namespace"
)

// serviceConnectPort defines a port exposed via ECS Service Connect.
type serviceConnectPort struct {
	portName      string
	port          int
	discoveryName string
}

type ecsConfig struct {
	cluster          string
	region           string
	subnetIDs        []string
	securityGroupIDs []string
	capacityProvider string
	configBucket     string
	serviceConnectNS string
}

func newEcsConfig(annotations map[string]string) (ecsConfig, error) {
	required := []string{
		annotationCluster,
		annotationRegion,
		annotationSubnetIDs,
		annotationSecurityGroupIDs,
		annotationConfigBucket,
		annotationServiceConnectNS,
	}

	if annotations == nil {
		return ecsConfig{}, fmt.Errorf("metadata.annotations is required, set %s",
			strings.Join(required, " and "))
	}

	for _, r := range required {
		val, ok := annotations[r]
		if !ok || val == "" {
			return ecsConfig{}, fmt.Errorf("required annotation %q is missing", r)
		}
	}

	return ecsConfig{
		cluster:          annotations[annotationCluster],
		region:           annotations[annotationRegion],
		subnetIDs:        strings.Split(annotations[annotationSubnetIDs], ","),
		securityGroupIDs: strings.Split(annotations[annotationSecurityGroupIDs], ","),
		capacityProvider: annotations[annotationCapacityProvider],
		configBucket:     annotations[annotationConfigBucket],
		serviceConnectNS: annotations[annotationServiceConnectNS],
	}, nil
}

func (c *ecsCasting) Cast(ctx context.Context, config v1alpha1.Casting, poursPath string) error {
	c.logger.InfoContext(ctx, "Deploying to AWS ECS")

	deploymentDir := filepath.Join(poursPath, rootcasting.DeploymentDir)

	cfg, err := newEcsConfig(config.Metadata.Annotations)
	if err != nil {
		return err
	}

	runctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// Upload config files to S3.
	if err := c.uploadConfigs(runctx, cfg, config, deploymentDir); err != nil {
		return fmt.Errorf("failed to upload configs to S3: %w", err)
	}

	// Deploy each service: register task definition, then create/update ECS service.
	type svcEntry struct {
		component  string
		taskDefDir string
		ports      []serviceConnectPort
	}
	svcs := []svcEntry{
		{"telemetrykeeper", filepath.Join("telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String()), []serviceConnectPort{
			{"keeper-client", 9181, "telemetrykeeper"},
			{"keeper-raft", 9234, "telemetrykeeper"},
		}},
		{"telemetrystore", filepath.Join("telemetrystore", config.Spec.TelemetryStore.Kind.String()), []serviceConnectPort{
			{"clickhouse-native", 9000, "telemetrystore"},
			{"clickhouse-http", 8123, "telemetrystore"},
		}},
		{"telemetrystore-migrator", "telemetrystore-migrator", nil},
		{"metastore", filepath.Join("metastore", config.Spec.MetaStore.Kind.String()), []serviceConnectPort{
			{"postgres", 5432, "metastore"},
		}},
		{"signoz", "signoz", []serviceConnectPort{
			{"signoz-http", 8080, "signoz"},
			{"signoz-opamp", 4320, "signoz"},
		}},
		{"ingester", "ingester", []serviceConnectPort{
			{"otel-grpc", 4317, "ingester"},
			{"otel-http", 4318, "ingester"},
		}},
	}

	for _, svc := range svcs {
		taskDefPath := filepath.Join(deploymentDir, svc.taskDefDir, "task-definition.json")
		if _, err := os.Stat(taskDefPath); os.IsNotExist(err) {
			continue
		}

		revision, err := c.registerTaskDefinition(runctx, cfg, taskDefPath)
		if err != nil {
			return fmt.Errorf("failed to register %s task definition: %w", svc.component, err)
		}

		serviceName := config.Metadata.Name + "-" + svc.component
		if err := c.createOrUpdateService(runctx, cfg, serviceName, revision, svc.ports); err != nil {
			return fmt.Errorf("failed to create/update %s service: %w", svc.component, err)
		}

		c.logger.InfoContext(runctx, "Deployed service",
			slog.String("component", svc.component),
			slog.String("service", serviceName),
		)
	}

	c.logger.InfoContext(ctx, "Deployment complete",
		slog.String("name", config.Metadata.Name),
		slog.String("cluster", cfg.cluster),
	)
	return nil
}

// --- AWS Helpers ---

func (c *ecsCasting) uploadConfigs(ctx context.Context, cfg ecsConfig, config v1alpha1.Casting, deploymentDir string) error {
	type componentConfig struct {
		component string
		kind      string
	}

	components := []componentConfig{
		{"telemetrykeeper", config.Spec.TelemetryKeeper.Kind.String()},
		{"telemetrystore", config.Spec.TelemetryStore.Kind.String()},
		{"metastore", config.Spec.MetaStore.Kind.String()},
	}

	for _, comp := range components {
		srcDir := filepath.Join(deploymentDir, comp.component, comp.kind)
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}
		s3Path := fmt.Sprintf("s3://%s/%s/%s/", cfg.configBucket, config.Metadata.Name, comp.component)
		if err := c.awsS3Sync(ctx, cfg, srcDir, s3Path); err != nil {
			return fmt.Errorf("failed to upload %s configs: %w", comp.component, err)
		}
	}

	// Ingester configs don't use Kind subdirectory — they sit alongside task-definition.json.
	ingesterDir := filepath.Join(deploymentDir, "ingester")
	if _, err := os.Stat(ingesterDir); os.IsNotExist(err) {
		return nil
	}
	s3Path := fmt.Sprintf("s3://%s/%s/ingester/", cfg.configBucket, config.Metadata.Name)
	if err := c.awsS3SyncExclude(ctx, cfg, ingesterDir, s3Path, "task-definition.json"); err != nil {
		return fmt.Errorf("failed to upload ingester configs: %w", err)
	}

	return nil
}

func (c *ecsCasting) awsS3Sync(ctx context.Context, cfg ecsConfig, srcDir, s3Path string) error {
	args := []string{
		"s3", "sync",
		srcDir, s3Path,
		"--region", cfg.region,
	}

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "aws "+strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws s3 sync failed: %w", err)
	}

	return nil
}

func (c *ecsCasting) awsS3SyncExclude(ctx context.Context, cfg ecsConfig, srcDir, s3Path, exclude string) error {
	args := []string{
		"s3", "sync",
		srcDir, s3Path,
		"--exclude", exclude,
		"--region", cfg.region,
	}

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "aws "+strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws s3 sync failed: %w", err)
	}

	return nil
}

func (c *ecsCasting) registerTaskDefinition(ctx context.Context, cfg ecsConfig, taskDefPath string) (string, error) {
	args := []string{
		"ecs", "register-task-definition",
		"--cli-input-json", "file://" + taskDefPath,
		"--region", cfg.region,
		"--output", "json",
	}

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "aws "+strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, "aws", args...)
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("aws ecs register-task-definition failed: %w", err)
	}

	var result struct {
		TaskDefinition struct {
			TaskDefinitionArn string `json:"taskDefinitionArn"`
			Revision          int    `json:"revision"`
		} `json:"taskDefinition"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse task definition output: %w", err)
	}

	c.logger.InfoContext(ctx, "registered task definition",
		slog.String("arn", result.TaskDefinition.TaskDefinitionArn),
		slog.Int("revision", result.TaskDefinition.Revision),
	)

	return result.TaskDefinition.TaskDefinitionArn, nil
}

// buildServiceConnectConfig builds the JSON for --service-connect-configuration.
func buildServiceConnectConfig(namespace string, ports []serviceConnectPort) string {
	type clientAlias struct {
		Port    int    `json:"port"`
		DNSName string `json:"dnsName"`
	}
	type scService struct {
		PortName      string        `json:"portName"`
		DiscoveryName string        `json:"discoveryName,omitempty"`
		ClientAliases []clientAlias `json:"clientAliases"`
	}
	type scConfig struct {
		Enabled   bool        `json:"enabled"`
		Namespace string      `json:"namespace"`
		Services  []scService `json:"services,omitempty"`
	}

	cfg := scConfig{
		Enabled:   true,
		Namespace: namespace,
	}

	for _, p := range ports {
		cfg.Services = append(cfg.Services, scService{
			PortName:      p.portName,
			DiscoveryName: p.discoveryName,
			ClientAliases: []clientAlias{{Port: p.port, DNSName: p.discoveryName}},
		})
	}

	data, _ := json.Marshal(cfg)
	return string(data)
}

func (c *ecsCasting) createOrUpdateService(ctx context.Context, cfg ecsConfig, serviceName, taskDefARN string, ports []serviceConnectPort) error {
	subnets := strings.Join(cfg.subnetIDs, ",")
	securityGroups := strings.Join(cfg.securityGroupIDs, ",")
	networkConfig := fmt.Sprintf("awsvpcConfiguration={subnets=[%s],securityGroups=[%s],assignPublicIp=DISABLED}", subnets, securityGroups)
	scConfig := buildServiceConnectConfig(cfg.serviceConnectNS, ports)

	// Try to update the service first; if it doesn't exist, create it.
	updateArgs := []string{
		"ecs", "update-service",
		"--cluster", cfg.cluster,
		"--service", serviceName,
		"--task-definition", taskDefARN,
		"--service-connect-configuration", scConfig,
		"--region", cfg.region,
		"--output", "none",
	}

	c.logger.DebugContext(ctx, "Attempting service update", slog.String("service", serviceName))

	updateCmd := exec.CommandContext(ctx, "aws", updateArgs...)
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr

	if err := updateCmd.Run(); err == nil {
		c.logger.InfoContext(ctx, "updated ECS service", slog.String("service", serviceName))
		return nil
	}

	// Service doesn't exist, create it.
	c.logger.InfoContext(ctx, "service does not exist, creating", slog.String("service", serviceName))

	createArgs := []string{
		"ecs", "create-service",
		"--cluster", cfg.cluster,
		"--service-name", serviceName,
		"--task-definition", taskDefARN,
		"--desired-count", "1",
		"--network-configuration", networkConfig,
		"--service-connect-configuration", scConfig,
		"--deployment-configuration", "deploymentCircuitBreaker={enable=true,rollback=true}",
		"--enable-execute-command",
		"--region", cfg.region,
		"--output", "none",
	}

	if cfg.capacityProvider != "" {
		createArgs = append(createArgs,
			"--capacity-provider-strategy", fmt.Sprintf("capacityProvider=%s,weight=1,base=0", cfg.capacityProvider),
		)
	} else {
		createArgs = append(createArgs, "--launch-type", "EC2")
	}

	c.logger.DebugContext(ctx, "Running command", slog.String("command", "aws "+strings.Join(createArgs, " ")))

	createCmd := exec.CommandContext(ctx, "aws", createArgs...)
	createCmd.Stdout = os.Stdout
	createCmd.Stderr = os.Stderr

	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("aws ecs create-service failed: %w", err)
	}

	c.logger.InfoContext(ctx, "created ECS service", slog.String("service", serviceName))
	return nil
}
