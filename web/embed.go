package web

import "embed"

var (
	//go:embed template/*
	TemplateFS embed.FS

	//go:embed static/*
	StaticFS embed.FS
)
