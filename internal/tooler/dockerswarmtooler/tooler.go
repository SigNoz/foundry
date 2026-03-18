package dockerswarmtooler

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	root "github.com/signoz/foundry/internal/tooler"
)

var _ root.Tooler = (*dockerSwarmTooler)(nil)

type dockerSwarmTooler struct{}

func New() *dockerSwarmTooler {
	return &dockerSwarmTooler{}
}

func (tooler *dockerSwarmTooler) Name() string {
	return "docker-swarm"
}

// Gauge checks that docker is available, swarm mode is active, and the local
// node is a manager (required for docker stack deploy).
func (tooler *dockerSwarmTooler) Gauge(ctx context.Context) error {
	if err := root.ExecChecker(ctx, "docker"); err != nil {
		return err
	}

	// Check swarm is active.
	stateCmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Swarm.LocalNodeState}}")
	stateOut, err := stateCmd.Output()
	if err != nil {
		return errors.New("failed to check docker swarm status: " + err.Error())
	}
	state := strings.TrimSpace(string(stateOut))
	if state != "active" {
		return errors.New("docker swarm is not active (state: " + state + "); run 'docker swarm init' to initialize")
	}

	// Verify the local node is a manager — stack deploy only works on managers.
	roleCmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.Swarm.ControlAvailable}}")
	roleOut, err := roleCmd.Output()
	if err != nil {
		return errors.New("failed to check docker swarm manager status: " + err.Error())
	}
	if strings.TrimSpace(string(roleOut)) != "true" {
		return errors.New("current node is a swarm worker, not a manager; 'docker stack deploy' must be run from a manager node")
	}

	return nil
}

func (tooler *dockerSwarmTooler) Install(ctx context.Context) error {
	return nil
}
