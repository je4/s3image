package server

import "embed"

//go:template template/index.gohtml
var templateFS embed.FS

var templateFiles = map[string]string{
	"index": "template/index.gohtml",
}
