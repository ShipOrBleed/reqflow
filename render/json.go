package render

import (
	"encoding/json"
	"io"

	reqflow "github.com/thzgajendra/reqflow"
)

type JSONRenderer struct{}

func (j *JSONRenderer) Render(g *reqflow.Graph, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(g)
}
