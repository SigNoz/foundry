package postgres

import (

	"errors"

	"cuelang.org/go/cue"
	"github.com/signoz/foundry/internal/common"
)

type Generator struct {}

func (g *Generator) GenerateComponent(config cue.Value) (map[string][]byte, error){

	files := make(map[string][]byte)

	postgresConfig := config.LookupPath(cue.ParsePath("components.postgres.config"))

	fields, err := postgresConfig.Fields()
	if err != nil {
		return nil, errors.New("failed to get postgres config fields:" + err.Error())
	}

	for fields.Next(){
		key := fields.Selector().String()
		value := fields.Value()
		
		switch key {
		case "auth":
			authConf, err := common.MapToEnv(value)
			if err != nil {
				return nil, errors.New("failed to convert auth config to env: " + err.Error())
			}
			files["auth.env"] = authConf
		default:
			conf, err := common.MapToINI(value)
			if err != nil {
				return nil, errors.New("failed to convert " + key + " config to ini: " + err.Error())
			}
			files[key + ".conf"] = conf
		}
	}
	return files, nil
}