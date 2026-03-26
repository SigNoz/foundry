package terraform

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl templates/aws/ec2/*.gotmpl templates/aws/eks/*.gotmpl templates/gcp/gce/*.gotmpl templates/gcp/gke/*.gotmpl templates/azure/vm/*.gotmpl templates/azure/aks/*.gotmpl
var templates embed.FS

// Common templates.
var (
	providersTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/providers.tf.json.gotmpl", types.FormatJSON)
)

// AWS EC2 templates.
var (
	awsEC2MainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/ec2/main.tf.json.gotmpl", types.FormatJSON)
	awsEC2VariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/ec2/variables.tf.json.gotmpl", types.FormatJSON)
	awsEC2OutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/ec2/outputs.tf.json.gotmpl", types.FormatJSON)
)

// AWS EKS templates.
var (
	awsEKSMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/eks/main.tf.json.gotmpl", types.FormatJSON)
	awsEKSVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/eks/variables.tf.json.gotmpl", types.FormatJSON)
	awsEKSOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/eks/outputs.tf.json.gotmpl", types.FormatJSON)
)

// GCP GCE templates.
var (
	gcpGCEMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gce/main.tf.json.gotmpl", types.FormatJSON)
	gcpGCEVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gce/variables.tf.json.gotmpl", types.FormatJSON)
	gcpGCEOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gce/outputs.tf.json.gotmpl", types.FormatJSON)
)

// GCP GKE templates.
var (
	gcpGKEMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gke/main.tf.json.gotmpl", types.FormatJSON)
	gcpGKEVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gke/variables.tf.json.gotmpl", types.FormatJSON)
	gcpGKEOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gke/outputs.tf.json.gotmpl", types.FormatJSON)
)

// Azure VM templates.
var (
	azureVMMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/vm/main.tf.json.gotmpl", types.FormatJSON)
	azureVMVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/vm/variables.tf.json.gotmpl", types.FormatJSON)
	azureVMOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/vm/outputs.tf.json.gotmpl", types.FormatJSON)
)

// Azure AKS templates.
var (
	azureAKSMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/aks/main.tf.json.gotmpl", types.FormatJSON)
	azureAKSVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/aks/variables.tf.json.gotmpl", types.FormatJSON)
	azureAKSOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/aks/outputs.tf.json.gotmpl", types.FormatJSON)
)
