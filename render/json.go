package render

import (
	"encoding/json"
	"io"

	"github.com/zopdev/govis"
)

type JSONRenderer struct{}

func (j *JSONRenderer) Render(g *structmap.Graph, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(g)
}
