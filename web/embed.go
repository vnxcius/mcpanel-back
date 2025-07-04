package web

import "embed"

//go:embed templates/* static/*
var TemplatesFS embed.FS
