package templates

import "embed"

// Append "**/*" if you also have template files in subdirectories
//
//go:embed *.html
var Templates embed.FS
