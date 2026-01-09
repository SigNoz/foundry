package foundry

import (
	"log/slog"

	"github.com/signoz/foundry/internal/casting"
	"github.com/signoz/foundry/internal/casting/dockercomposecasting"
)

func NewCastings(logger *slog.Logger) map[string]casting.Casting {
	return map[string]casting.Casting{
		"docker": dockercomposecasting.New(logger),
	}
}
