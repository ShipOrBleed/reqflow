package govis

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// OTLPExport represents the top-level structure of an OTLP JSON export.
type OTLPExport struct {
	ResourceSpans []resourceSpan `json:"resourceSpans"`
}

type resourceSpan struct {
	Resource  resource    `json:"resource"`
	ScopeSpans []scopeSpan `json:"scopeSpans"`
}

type resource struct {
	Attributes []attribute `json:"attributes"`
}

type scopeSpan struct {
	Spans []span `json:"spans"`
}

type span struct {
	Name            string      `json:"name"`
	Kind            int         `json:"kind"`
	StartTimeUnixNano string   `json:"startTimeUnixNano"`
	EndTimeUnixNano   string   `json:"endTimeUnixNano"`
	Attributes      []attribute `json:"attributes"`
	Status          *spanStatus `json:"status"`
	ParentSpanID    string      `json:"parentSpanId"`
	SpanID          string      `json:"spanId"`
	TraceID         string      `json:"traceId"`
}

type attribute struct {
	Key   string         `json:"key"`
	Value attributeValue `json:"value"`
}

type attributeValue struct {
	StringValue string `json:"stringValue"`
	IntValue    string `json:"intValue"`
}

type spanStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// spanMetrics aggregates span data for a single operation.
type spanMetrics struct {
	OperationName string
	ServiceName   string
	Durations     []float64 // in milliseconds
	ErrorCount    int
	TotalCount    int
}

// ExtractOtelTrace parses an OTLP JSON export file and maps span operations
// to graph nodes, tagging them with latency and error metrics.
func ExtractOtelTrace(filePath string, graph *Graph) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	var export OTLPExport
	if err := json.Unmarshal(data, &export); err != nil {
		return
	}

	// Aggregate metrics per operation
	metrics := make(map[string]*spanMetrics)

	for _, rs := range export.ResourceSpans {
		serviceName := extractServiceName(rs.Resource)

		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				key := fmt.Sprintf("%s/%s", serviceName, s.Name)

				if _, exists := metrics[key]; !exists {
					metrics[key] = &spanMetrics{
						OperationName: s.Name,
						ServiceName:   serviceName,
					}
				}
				m := metrics[key]
				m.TotalCount++

				// Calculate duration
				dur := spanDurationMs(s)
				if dur > 0 {
					m.Durations = append(m.Durations, dur)
				}

				// Check for errors
				if s.Status != nil && s.Status.Code == 2 {
					m.ErrorCount++
				}
			}
		}
	}

	// Map metrics to graph nodes
	for _, m := range metrics {
		nodeID := matchSpanToNode(m, graph)
		if nodeID == "" {
			continue
		}

		node := graph.Nodes[nodeID]
		if node == nil {
			continue
		}

		// Calculate percentiles
		if len(m.Durations) > 0 {
			sort.Float64s(m.Durations)
			avg := average(m.Durations)
			p99 := percentile(m.Durations, 99)
			node.Meta["otel_avg_duration"] = fmt.Sprintf("%.1fms", avg)
			node.Meta["otel_p99"] = fmt.Sprintf("%.1fms", p99)
			node.Meta["otel_call_count"] = fmt.Sprintf("%d", m.TotalCount)
		}

		if m.TotalCount > 0 {
			errRate := float64(m.ErrorCount) / float64(m.TotalCount) * 100
			node.Meta["otel_error_rate"] = fmt.Sprintf("%.1f%%", errRate)
		}

		node.Meta["otel_service"] = m.ServiceName
	}
}

func extractServiceName(r resource) string {
	for _, attr := range r.Attributes {
		if attr.Key == "service.name" {
			return attr.Value.StringValue
		}
	}
	return "unknown"
}

func spanDurationMs(s span) float64 {
	// Parse nanosecond timestamps
	var start, end int64
	fmt.Sscanf(s.StartTimeUnixNano, "%d", &start)
	fmt.Sscanf(s.EndTimeUnixNano, "%d", &end)
	if start > 0 && end > start {
		return float64(end-start) / 1e6 // ns → ms
	}
	return 0
}

// matchSpanToNode finds the best matching graph node for a span operation.
func matchSpanToNode(m *spanMetrics, graph *Graph) string {
	opName := m.OperationName

	// Try exact match on route metadata
	for id, node := range graph.Nodes {
		if route, ok := node.Meta["route"]; ok {
			if strings.Contains(route, opName) || strings.Contains(opName, route) {
				return id
			}
		}
	}

	// Try match on node name
	for id, node := range graph.Nodes {
		if strings.EqualFold(node.Name, opName) {
			return id
		}
		// Try partial match (e.g., "GetUser" matches "UserHandler.GetUser")
		if strings.HasSuffix(strings.ToLower(node.Name), strings.ToLower(opName)) {
			return id
		}
	}

	// Try match on function name in methods
	for id, node := range graph.Nodes {
		for _, method := range node.Methods {
			if strings.EqualFold(method, opName) {
				return id
			}
		}
	}

	return ""
}

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func percentile(sorted []float64, pct float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(pct/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
