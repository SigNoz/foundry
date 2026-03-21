package wizard

import (
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/internal/types"
)

// DeploymentOption represents a selectable deployment target.
type DeploymentOption struct {
	Label      string
	Deployment v1alpha1.TypeDeployment
}

// Result holds the wizard output.
type Result struct {
	Name       string
	Deployment v1alpha1.TypeDeployment
	OutputPath string
}

// Run executes the interactive init wizard.
func Run(deployments []v1alpha1.TypeDeployment) (*Result, error) {
	options := buildDeploymentOptions(deployments)

	var (
		selectedIdx int
		name        string
		outputPath  string
	)

	// Build huh select options
	selectOptions := make([]huh.Option[int], len(options))
	for i, opt := range options {
		selectOptions[i] = huh.NewOption(opt.Label, i)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Deployment target").
				Description("Select how you want to deploy SigNoz").
				Options(selectOptions...).
				Value(&selectedIdx),

			huh.NewInput().
				Title("Installation name").
				Description("A name to identify this SigNoz installation").
				Placeholder("signoz").
				Value(&name),

			huh.NewInput().
				Title("Output path").
				Description("Where to write the casting.yaml file").
				Placeholder("casting.yaml").
				Value(&outputPath),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard cancelled: %w", err)
	}

	// Apply defaults for empty values
	if name == "" {
		name = "signoz"
	}
	if outputPath == "" {
		outputPath = "casting.yaml"
	}

	return &Result{
		Name:       name,
		Deployment: options[selectedIdx].Deployment,
		OutputPath: outputPath,
	}, nil
}

// WriteCasting generates a casting.yaml from the wizard result and writes it to disk.
func WriteCasting(result *Result) error {
	casting := v1alpha1.DefaultCasting()
	casting.Metadata.Name = result.Name
	casting.Spec.Deployment = result.Deployment

	data := types.MustMarshalYAML(casting)
	return os.WriteFile(result.OutputPath, data, 0644)
}

// BuildCastingFromFlags creates a Result from CLI flags (non-interactive mode).
func BuildCastingFromFlags(name, mode, flavor, platform, outputPath string) (*Result, error) {
	if mode == "" && platform == "" {
		return nil, fmt.Errorf("--mode or --platform is required in non-interactive mode")
	}
	if flavor == "" {
		return nil, fmt.Errorf("--flavor is required in non-interactive mode")
	}

	if name == "" {
		name = "signoz"
	}
	if outputPath == "" {
		outputPath = "casting.yaml"
	}

	return &Result{
		Name: name,
		Deployment: v1alpha1.TypeDeployment{
			Platform: platform,
			Mode:     mode,
			Flavor:   flavor,
		},
		OutputPath: outputPath,
	}, nil
}

func buildDeploymentOptions(deployments []v1alpha1.TypeDeployment) []DeploymentOption {
	options := make([]DeploymentOption, 0, len(deployments))
	for _, d := range deployments {
		label := formatDeploymentLabel(d)
		options = append(options, DeploymentOption{
			Label:      label,
			Deployment: d,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	return options
}

func formatDeploymentLabel(d v1alpha1.TypeDeployment) string {
	if d.Platform != "" && d.Mode != "" {
		return fmt.Sprintf("%s / %s / %s", d.Platform, d.Mode, d.Flavor)
	}
	if d.Platform != "" {
		return fmt.Sprintf("%s / %s", d.Platform, d.Flavor)
	}
	return fmt.Sprintf("%s / %s", d.Mode, d.Flavor)
}
