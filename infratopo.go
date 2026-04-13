package govis

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtractInfraTopo scans for Dockerfiles, docker-compose.yml, and Kubernetes
// manifests to build a KindContainer topology with dependency edges.
func ExtractInfraTopo(dir string, graph *Graph) {
	workDir := dir
	if workDir == "./..." || workDir == "" {
		workDir = "."
	}

	// Scan for infrastructure files
	filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()

		switch {
		case name == "Dockerfile" || strings.HasPrefix(name, "Dockerfile."):
			parseDockerfile(path, graph)
		case name == "docker-compose.yml" || name == "docker-compose.yaml":
			parseDockerCompose(path, graph)
		case (strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) &&
			isK8sManifest(path):
			parseK8sManifest(path, graph)
		}
		return nil
	})
}

// parseDockerfile extracts FROM, EXPOSE, and CMD from a Dockerfile.
func parseDockerfile(path string, graph *Graph) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	containerID := fmt.Sprintf("container.dockerfile.%s", sanitizePath(path))
	node := &Node{
		ID:      containerID,
		Kind:    KindContainer,
		Name:    filepath.Base(filepath.Dir(path)),
		Package: "infrastructure",
		File:    path,
		Line:    1,
		Meta:    map[string]string{"source": "Dockerfile"},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "FROM "):
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				node.Meta["base_image"] = parts[1]
			}
		case strings.HasPrefix(upper, "EXPOSE "):
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				existing := node.Meta["ports"]
				if existing != "" {
					existing += ", "
				}
				node.Meta["ports"] = existing + strings.Join(parts[1:], ", ")
			}
		case strings.HasPrefix(upper, "CMD ") || strings.HasPrefix(upper, "ENTRYPOINT "):
			node.Meta["entrypoint"] = line
		}
	}

	graph.AddNode(node)
}

// dockerComposeService represents a service in docker-compose.yml.
type dockerComposeService struct {
	Image      string   `yaml:"image"`
	Build      any      `yaml:"build"`
	Ports      []string `yaml:"ports"`
	DependsOn  any      `yaml:"depends_on"`
	Networks   []string `yaml:"networks"`
	Volumes    []string `yaml:"volumes"`
	Command    string   `yaml:"command"`
	Entrypoint string   `yaml:"entrypoint"`
}

type dockerComposeFile struct {
	Services map[string]dockerComposeService `yaml:"services"`
}

// parseDockerCompose extracts services, ports, depends_on from docker-compose.yml.
func parseDockerCompose(path string, graph *Graph) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var compose dockerComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return
	}

	serviceIDs := make(map[string]string) // service name → node ID

	for name, svc := range compose.Services {
		containerID := fmt.Sprintf("container.compose.%s", name)
		serviceIDs[name] = containerID

		node := &Node{
			ID:      containerID,
			Kind:    KindContainer,
			Name:    name,
			Package: "infrastructure",
			File:    path,
			Line:    1,
			Meta: map[string]string{
				"source": "docker-compose",
			},
		}

		if svc.Image != "" {
			node.Meta["image"] = svc.Image
		}
		if len(svc.Ports) > 0 {
			node.Meta["ports"] = strings.Join(svc.Ports, ", ")
		}
		if len(svc.Networks) > 0 {
			node.Meta["networks"] = strings.Join(svc.Networks, ", ")
		}

		graph.AddNode(node)
	}

	// Create dependency edges
	for name, svc := range compose.Services {
		fromID := serviceIDs[name]
		deps := extractDependsOn(svc.DependsOn)
		for _, dep := range deps {
			if toID, exists := serviceIDs[dep]; exists {
				graph.AddEdge(fromID, toID, EdgeDepends)
			}
		}
	}
}

// extractDependsOn handles both array and map forms of depends_on.
func extractDependsOn(v any) []string {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []any:
		var deps []string
		for _, item := range val {
			if s, ok := item.(string); ok {
				deps = append(deps, s)
			}
		}
		return deps
	case map[string]any:
		var deps []string
		for k := range val {
			deps = append(deps, k)
		}
		return deps
	}
	return nil
}

// k8sManifest is a minimal representation of a Kubernetes manifest.
type k8sManifest struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Template struct {
			Spec struct {
				Containers []struct {
					Name  string `yaml:"name"`
					Image string `yaml:"image"`
					Ports []struct {
						ContainerPort int `yaml:"containerPort"`
					} `yaml:"ports"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
		// For Service kind
		ServicePorts []struct {
			Port       int    `yaml:"port"`
			TargetPort int    `yaml:"targetPort"`
			Protocol   string `yaml:"protocol"`
		} `yaml:"ports"`
		Selector map[string]string `yaml:"selector"`
	} `yaml:"spec"`
}

func isK8sManifest(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	content := string(data)
	return strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:")
}

// parseK8sManifest extracts Deployment/Service/Ingress info.
func parseK8sManifest(path string, graph *Graph) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Handle multi-document YAML
	docs := strings.Split(string(data), "---")
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var manifest k8sManifest
		if err := yaml.Unmarshal([]byte(doc), &manifest); err != nil {
			continue
		}

		if manifest.Kind == "" || manifest.Metadata.Name == "" {
			continue
		}

		switch manifest.Kind {
		case "Deployment", "StatefulSet", "DaemonSet":
			for _, container := range manifest.Spec.Template.Spec.Containers {
				containerID := fmt.Sprintf("container.k8s.%s.%s", manifest.Metadata.Name, container.Name)
				node := &Node{
					ID:      containerID,
					Kind:    KindContainer,
					Name:    fmt.Sprintf("%s/%s", manifest.Metadata.Name, container.Name),
					Package: "infrastructure",
					File:    path,
					Line:    1,
					Meta: map[string]string{
						"source":    "kubernetes",
						"k8s_kind":  manifest.Kind,
						"image":     container.Image,
						"namespace": manifest.Metadata.Namespace,
					},
				}
				var ports []string
				for _, p := range container.Ports {
					ports = append(ports, fmt.Sprintf("%d", p.ContainerPort))
				}
				if len(ports) > 0 {
					node.Meta["ports"] = strings.Join(ports, ", ")
				}
				graph.AddNode(node)
			}

		case "Service":
			containerID := fmt.Sprintf("container.k8s.svc.%s", manifest.Metadata.Name)
			node := &Node{
				ID:      containerID,
				Kind:    KindContainer,
				Name:    fmt.Sprintf("svc/%s", manifest.Metadata.Name),
				Package: "infrastructure",
				File:    path,
				Line:    1,
				Meta: map[string]string{
					"source":    "kubernetes",
					"k8s_kind":  "Service",
					"namespace": manifest.Metadata.Namespace,
				},
			}
			var ports []string
			for _, p := range manifest.Spec.ServicePorts {
				ports = append(ports, fmt.Sprintf("%d→%d", p.Port, p.TargetPort))
			}
			if len(ports) > 0 {
				node.Meta["ports"] = strings.Join(ports, ", ")
			}
			graph.AddNode(node)

			// Link service to matching deployment containers via selector
			if len(manifest.Spec.Selector) > 0 {
				for _, existing := range graph.Nodes {
					if existing.Kind == KindContainer && strings.Contains(existing.ID, "container.k8s.") &&
						!strings.Contains(existing.ID, "container.k8s.svc.") {
						// Simple name-based matching
						if strings.Contains(existing.Name, manifest.Metadata.Name) {
							graph.AddEdge(containerID, existing.ID, EdgeDepends)
						}
					}
				}
			}
		}
	}
}

func sanitizePath(path string) string {
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, ".", "_")
	path = strings.ReplaceAll(path, "-", "_")
	return path
}
