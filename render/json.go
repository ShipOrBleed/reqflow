package render

import (
	"encoding/json"
	"io"

	govis "github.com/zopdev/govis"
)

type JSONRenderer struct{}

func (j *JSONRenderer) Render(g *govis.Graph, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(g)
}
