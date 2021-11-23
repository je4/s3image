package server

import "embed"

//go:embed template/index.gohtml
var templateFS embed.FS

var templateFiles = map[string]string{
	"index": "template/index.gohtml",
}
