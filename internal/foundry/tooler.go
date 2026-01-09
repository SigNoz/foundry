package foundry

import (
	"github.com/signoz/foundry/internal/tooler"
	"github.com/signoz/foundry/internal/tooler/dockercomposetooler"
	"github.com/signoz/foundry/internal/tooler/dockertooler"
)

var DeploymentModeToTooler = map[string][]tooler.Tooler{
	"docker": {dockertooler.New(), dockercomposetooler.New()},
}
