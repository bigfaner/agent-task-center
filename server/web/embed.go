// Package web embeds the built React application assets.
// The web UI must be built first (cd web && npm run build) so that
// the web/dist directory exists relative to this file.
package web

import "embed"

// DistFS holds the embedded files from web/dist.
//
//go:embed dist
var DistFS embed.FS
