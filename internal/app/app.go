package app

import (
	"eleven-hacks/internal/config"
	"embed"
	"golang.org/x/net/context"
)

func GetConfig(ctx context.Context) *config.Config {
	if ctx == nil {
		return nil
	}
	if config, ok := ctx.Value("config").(*config.Config); ok {
		return config
	}
	return nil
}

func GetAssets(ctx context.Context) *embed.FS {
	if ctx == nil {
		return nil
	}
	if assets, ok := ctx.Value("assets").(*embed.FS); ok {
		return assets
	}
	return nil
}
