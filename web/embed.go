// Package web embarque les gabarits HTML et les fichiers statiques dans le binaire

package web

import "embed"

// Files contient les gabarits (templates/) et les fichiers statiques (static/)
//
//go:embed all:templates static
var Files embed.FS
