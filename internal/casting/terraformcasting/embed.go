package terraformcasting

import (
	"embed"

	"github.com/signoz/foundry/internal/types"
)

//go:embed templates/*.gotmpl templates/aws/ec2/*.gotmpl templates/aws/eks/*.gotmpl templates/gcp/gce/*.gotmpl templates/gcp/gke/*.gotmpl templates/azure/vm/*.gotmpl templates/azure/aks/*.gotmpl
var templates embed.FS

// Common templates.
var (
	providersTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/providers.tf.gotmpl", types.FormatHCL)
)

// AWS EC2 templates.
var (
	awsEC2MainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/ec2/main.tf.gotmpl", types.FormatHCL)
	awsEC2VariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/ec2/variables.tf.gotmpl", types.FormatHCL)
	awsEC2OutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/ec2/outputs.tf.gotmpl", types.FormatHCL)
)

// AWS EKS templates.
var (
	awsEKSMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/eks/main.tf.gotmpl", types.FormatHCL)
	awsEKSVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/eks/variables.tf.gotmpl", types.FormatHCL)
	awsEKSOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/aws/eks/outputs.tf.gotmpl", types.FormatHCL)
)

// GCP GCE templates.
var (
	gcpGCEMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gce/main.tf.gotmpl", types.FormatHCL)
	gcpGCEVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gce/variables.tf.gotmpl", types.FormatHCL)
	gcpGCEOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gce/outputs.tf.gotmpl", types.FormatHCL)
)

// GCP GKE templates.
var (
	gcpGKEMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gke/main.tf.gotmpl", types.FormatHCL)
	gcpGKEVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gke/variables.tf.gotmpl", types.FormatHCL)
	gcpGKEOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/gcp/gke/outputs.tf.gotmpl", types.FormatHCL)
)

// Azure VM templates.
var (
	azureVMMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/vm/main.tf.gotmpl", types.FormatHCL)
	azureVMVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/vm/variables.tf.gotmpl", types.FormatHCL)
	azureVMOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/vm/outputs.tf.gotmpl", types.FormatHCL)
)

// Azure AKS templates.
var (
	azureAKSMainTFTemplate      *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/aks/main.tf.gotmpl", types.FormatHCL)
	azureAKSVariablesTFTemplate *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/aks/variables.tf.gotmpl", types.FormatHCL)
	azureAKSOutputsTFTemplate   *types.Template = types.MustNewTemplateFromFS(templates, "templates/azure/aks/outputs.tf.gotmpl", types.FormatHCL)
)
