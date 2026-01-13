package linuxcasting

import (
	"bytes"
	"testing"

	"github.com/signoz/foundry/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestNotEmptyAndValid(t *testing.T) {
	serviceTemplates := map[string]*types.Template{
		"telemetryStoreServiceTemplate":  telemetryStoreServiceTemplate,
		"telemetryKeeperServiceTemplate": telemetryKeeperServiceTemplate,
		"metaStoreServiceTemplate":       metaStoreServiceTemplate,
		"signozServiceTemplate":          signozServiceTemplate,
		"ingesterServiceTemplate":        ingesterServiceTemplate,
	}

	for name, st := range serviceTemplates {
		assert.NotEmpty(t, st, "%s should not be empty", name)
		buf := bytes.NewBuffer(nil)
		err := st.Execute(buf, nil)
		assert.NoError(t, err, "error executing %s", name)
		assert.NotEmpty(t, buf.String(), "%s output should not be empty", name)
	}
}
