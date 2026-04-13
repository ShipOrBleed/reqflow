package render

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	govis "github.com/zopdev/govis"
)

// ExcalidrawRenderer generates an Excalidraw JSON file (.excalidraw)
// with nodes as colored rectangles and edges as arrows.
type ExcalidrawRenderer struct{}

type excalidrawFile struct {
	Type       string             `json:"type"`
	Version    int                `json:"version"`
	Source     string             `json:"source"`
	Elements   []excalidrawElement `json:"elements"`
	AppState   map[string]any     `json:"appState"`
}

type excalidrawElement struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	Width           float64 `json:"width,omitempty"`
	Height          float64 `json:"height,omitempty"`
	StrokeColor     string  `json:"strokeColor"`
	BackgroundColor string  `json:"backgroundColor"`
	FillStyle       string  `json:"fillStyle"`
	StrokeWidth     int     `json:"strokeWidth"`
	Roughness       int     `json:"roughness"`
	Opacity         int     `json:"opacity"`
	Text            string  `json:"text,omitempty"`
	FontSize        int     `json:"fontSize,omitempty"`
	FontFamily      int     `json:"fontFamily,omitempty"`
	TextAlign       string  `json:"textAlign,omitempty"`
	VerticalAlign   string  `json:"verticalAlign,omitempty"`
	ContainerID     string  `json:"containerId,omitempty"`
	BoundElements   []bound `json:"boundElements,omitempty"`
	Points          [][]float64 `json:"points,omitempty"`
	StartBinding    *binding `json:"startBinding,omitempty"`
	EndBinding      *binding `json:"endBinding,omitempty"`
	StartArrowhead  *string  `json:"startArrowhead"`
	EndArrowhead    *string  `json:"endArrowhead,omitempty"`
}

type bound struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type binding struct {
	ElementID string  `json:"elementId"`
	Focus     float64 `json:"focus"`
	Gap       int     `json:"gap"`
}

var excalidrawColors = map[string]string{
	"handler":    "#a5d8ff",
	"service":    "#b2f2bb",
	"store":      "#ffec99",
	"model":      "#ffc9c9",
	"event":      "#e9ecef",
	"middleware":  "#fff3bf",
	"grpc":       "#c5f6fa",
	"infra":      "#d0bfff",
	"route":      "#99e9f2",
	"envvar":     "#96f2d7",
	"table":      "#ffd8a8",
	"dependency": "#dee2e6",
	"container":  "#eebefa",
	"proto_rpc":  "#bac8ff",
	"proto_msg":  "#fcc2d7",
}

func (e *ExcalidrawRenderer) Render(g *govis.Graph, w io.Writer) error {
	var elements []excalidrawElement
	nodePositions := make(map[string][2]float64) // node ID → [x, y]

	// Sort nodes by package for grid layout
	type nodeEntry struct {
		id   string
		node *govis.Node
	}
	var sorted []nodeEntry
	for id, node := range g.Nodes {
		sorted = append(sorted, nodeEntry{id, node})
	}
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].node.Package != sorted[j].node.Package {
			return sorted[i].node.Package < sorted[j].node.Package
		}
		return sorted[i].id < sorted[j].id
	})

	// Grid layout: 5 columns
	cols := 5
	cellW, cellH := 260.0, 100.0
	gapX, gapY := 40.0, 40.0

	for i, entry := range sorted {
		col := i % cols
		row := i / cols
		x := float64(col) * (cellW + gapX)
		y := float64(row) * (cellH + gapY)

		nodePositions[entry.id] = [2]float64{x, y}

		color := excalidrawColors[string(entry.node.Kind)]
		if color == "" {
			color = "#dee2e6"
		}

		rectID := genID()
		textID := genID()

		label := fmt.Sprintf("%s\n[%s]", entry.node.Name, entry.node.Kind)

		// Rectangle
		elements = append(elements, excalidrawElement{
			ID:              rectID,
			Type:            "rectangle",
			X:               x,
			Y:               y,
			Width:           cellW,
			Height:          cellH,
			StrokeColor:     "#1e1e1e",
			BackgroundColor: color,
			FillStyle:       "solid",
			StrokeWidth:     2,
			Roughness:       1,
			Opacity:         100,
			BoundElements:   []bound{{ID: textID, Type: "text"}},
		})

		// Text label
		elements = append(elements, excalidrawElement{
			ID:              textID,
			Type:            "text",
			X:               x + 10,
			Y:               y + 20,
			Width:           cellW - 20,
			Height:          cellH - 40,
			StrokeColor:     "#1e1e1e",
			BackgroundColor: "transparent",
			FillStyle:       "solid",
			StrokeWidth:     1,
			Roughness:       0,
			Opacity:         100,
			Text:            label,
			FontSize:        16,
			FontFamily:      1,
			TextAlign:       "center",
			VerticalAlign:   "middle",
			ContainerID:     rectID,
		})
	}

	// Edges as arrows
	for _, edge := range g.Edges {
		fromPos, fromOK := nodePositions[edge.From]
		toPos, toOK := nodePositions[edge.To]
		if !fromOK || !toOK {
			continue
		}

		startX := fromPos[0] + cellW/2
		startY := fromPos[1] + cellH
		endX := toPos[0] + cellW/2
		endY := toPos[1]

		arrowhead := "arrow"
		elements = append(elements, excalidrawElement{
			ID:              genID(),
			Type:            "arrow",
			X:               startX,
			Y:               startY,
			Width:           endX - startX,
			Height:          endY - startY,
			StrokeColor:     "#495057",
			BackgroundColor: "transparent",
			FillStyle:       "solid",
			StrokeWidth:     1,
			Roughness:       1,
			Opacity:         100,
			Points:          [][]float64{{0, 0}, {endX - startX, endY - startY}},
			StartArrowhead:  nil,
			EndArrowhead:    &arrowhead,
		})
	}

	file := excalidrawFile{
		Type:     "excalidraw",
		Version:  2,
		Source:    "govis",
		Elements: elements,
		AppState: map[string]any{
			"gridSize":        20,
			"viewBackgroundColor": "#ffffff",
		},
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(file)
}

func genID() string {
	b := make([]byte, 10)
	rand.Read(b)
	return hex.EncodeToString(b)
}
