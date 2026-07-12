package ui

import "embed"

var (
	// TemplateFS holds HTML templates embedded at build time.
	//go:embed template/*
	TemplateFS embed.FS

	// StaticFS holds CSS and other static assets embedded at build time.
	//go:embed static/*
	StaticFS embed.FS
)
