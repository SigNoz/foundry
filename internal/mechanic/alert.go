package mechanic

import (
	"encoding/json"

	"github.com/tidwall/gjson"
)

// Alert is the metadata mechanic surfaces for a SigNoz alert rule. Data is the
// raw rule JSON as stored in the metastore's rule.data column; Name is decoded
// from it.
type Alert struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// MarshalJSON lets Alert satisfy json.Marshaler so it can be streamed to stdout
// via writer.WriteOutput. The alias breaks the method recursion.
func (a Alert) MarshalJSON() ([]byte, error) {
	type alias Alert
	return json.Marshal(alias(a))
}

// decodeAlert builds an Alert from a metastore row's id and raw rule JSON,
// extracting the human-readable name from the rule payload.
func decodeAlert(id string, data []byte) Alert {
	return Alert{
		ID:   id,
		Name: alertName(data),
		Data: json.RawMessage(data),
	}
}

// alertName pulls the rule's display name from its JSON payload, tolerating the
// handful of keys SigNoz has used across versions.
func alertName(data []byte) string {
	for _, key := range []string{"alert", "alertName", "name"} {
		if r := gjson.GetBytes(data, key); r.Exists() {
			if s := r.String(); s != "" {
				return s
			}
		}
	}
	return ""
}
