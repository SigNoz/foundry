package terraform

import (
	"embed"

	"github.com/signoz/foundry/internal/domain"
)

//go:embed templates/*.gotmpl templates/aws/ec2/*.gotmpl templates/aws/eks/*.gotmpl templates/gcp/gce/*.gotmpl templates/gcp/gke/*.gotmpl templates/azure/vm/*.gotmpl templates/azure/aks/*.gotmpl
var templates embed.FS

// Common templates.
var (
	providersTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/providers.tf.json.gotmpl", domain.FormatJSON)
)

// AWS EC2 templates.
var (
	awsEC2MainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/aws/ec2/main.tf.json.gotmpl", domain.FormatJSON)
	awsEC2VariablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/aws/ec2/variables.tf.json.gotmpl", domain.FormatJSON)
	awsEC2OutputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/aws/ec2/outputs.tf.json.gotmpl", domain.FormatJSON)
)

// AWS EKS templates.
var (
	awsEKSMainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/aws/eks/main.tf.json.gotmpl", domain.FormatJSON)
	awsEKSVariablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/aws/eks/variables.tf.json.gotmpl", domain.FormatJSON)
	awsEKSOutputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/aws/eks/outputs.tf.json.gotmpl", domain.FormatJSON)
)

// GCP GCE templates.
var (
	gcpGCEMainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/gcp/gce/main.tf.json.gotmpl", domain.FormatJSON)
	gcpGCEVariablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/gcp/gce/variables.tf.json.gotmpl", domain.FormatJSON)
	gcpGCEOutputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/gcp/gce/outputs.tf.json.gotmpl", domain.FormatJSON)
)

// GCP GKE templates.
var (
	gcpGKEMainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/gcp/gke/main.tf.json.gotmpl", domain.FormatJSON)
	gcpGKEVariablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/gcp/gke/variables.tf.json.gotmpl", domain.FormatJSON)
	gcpGKEOutputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/gcp/gke/outputs.tf.json.gotmpl", domain.FormatJSON)
)

// Azure VM templates.
var (
	azureVMMainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/azure/vm/main.tf.json.gotmpl", domain.FormatJSON)
	azureVMVariablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/azure/vm/variables.tf.json.gotmpl", domain.FormatJSON)
	azureVMOutputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/azure/vm/outputs.tf.json.gotmpl", domain.FormatJSON)
)

// Azure AKS templates.
var (
	azureAKSMainTFTemplate      *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/azure/aks/main.tf.json.gotmpl", domain.FormatJSON)
	azureAKSVariablesTFTemplate *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/azure/aks/variables.tf.json.gotmpl", domain.FormatJSON)
	azureAKSOutputsTFTemplate   *domain.Template = domain.MustNewTemplateFromFS(templates, "templates/azure/aks/outputs.tf.json.gotmpl", domain.FormatJSON)
)
