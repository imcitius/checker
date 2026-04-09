// SPDX-License-Identifier: BUSL-1.1

package web

import "embed"

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

//go:embed all:spa
var spaFS embed.FS
