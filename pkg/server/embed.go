package server

import "embed"

//go:embed template/index.gohtml
//go:embed template/pamphlet.gohtml
var templateFS embed.FS

var templateFiles = map[string]string{
	"index":    "template/index.gohtml",
	"pamphlet": "template/pamphlet.gohtml",
}
