package helm

import (
	"archive/zip"
	"bytes"
	"fmt"
	"text/template"

	"github.com/thatrajaryan/web-server/api/models"
)

type HelmGenerator struct {
	ProjectID   string
	ProjectName string
	Nodes       []models.Node
	Connections []models.Connection
}

func (g *HelmGenerator) Generate() ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Chart.yaml
	chartYaml := fmt.Sprintf(`apiVersion: v2
name: %s
description: A generated helm chart for %s
type: application
version: 0.1.0
appVersion: "1.0.0"
`, g.ProjectName, g.ProjectName)
	if err := g.addToZip(zw, g.ProjectName+"/Chart.yaml", chartYaml); err != nil {
		return nil, err
	}

	// Values.yaml
	valuesYaml := g.generateValues()
	if err := g.addToZip(zw, g.ProjectName+"/values.yaml", valuesYaml); err != nil {
		return nil, err
	}

	// Templates
	for _, node := range g.Nodes {
		tmpl, err := g.generateDeployment(node)
		if err != nil {
			return nil, err
		}
		if err := g.addToZip(zw, fmt.Sprintf("%s/templates/deployment-%s.yaml", g.ProjectName, node.ID), tmpl); err != nil {
			return nil, err
		}

		svc, err := g.generateService(node)
		if err != nil {
			return nil, err
		}
		if err := g.addToZip(zw, fmt.Sprintf("%s/templates/service-%s.yaml", g.ProjectName, node.ID), svc); err != nil {
			return nil, err
		}
	}

	// Hooks (ConfigMap and Proxy)
	for i, conn := range g.Connections {
		if conn.HookCode != "" {
			cm, err := g.generateHookConfigMap(conn, i)
			if err != nil {
				return nil, err
			}
			if err := g.addToZip(zw, fmt.Sprintf("%s/templates/hook-configmap-%d.yaml", g.ProjectName, i), cm); err != nil {
				return nil, err
			}

			proxy, err := g.generateHookProxy(conn, i)
			if err != nil {
				return nil, err
			}
			if err := g.addToZip(zw, fmt.Sprintf("%s/templates/hook-proxy-%d.yaml", g.ProjectName, i), proxy); err != nil {
				return nil, err
			}
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (g *HelmGenerator) addToZip(zw *zip.Writer, name, content string) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
}

func (g *HelmGenerator) generateValues() string {
	var buf bytes.Buffer
	buf.WriteString("global:\n  environment: production\n\n")
	for _, node := range g.Nodes {
		buf.WriteString(fmt.Sprintf("%s:\n", node.ID))
		buf.WriteString(fmt.Sprintf("  type: %s\n", node.Type))
		// Serialize config fields into values
		for k, v := range node.Config {
			buf.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

func (g *HelmGenerator) generateDeployment(node models.Node) (string, error) {
	image := "nginx:latest" // Default
	if node.Type == "server" {
		if img, ok := node.Config["image"].(string); ok && img != "" {
			image = img
		}
	} else if node.Type == "ai" {
		image = "infrastructure/ai-proxy:latest"
	} else if node.Type == "api-gateway" {
		image = "infrastructure/api-gateway:latest"
	}

	tmpl := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .ID }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: {{ .ID }}
  template:
    metadata:
      labels:
        app: {{ .ID }}
    spec:
      containers:
      - name: main
        image: {{ .Image }}
        ports:
        - containerPort: {{ .Port }}
        env:
        {{- range $key, $value := .Env }}
        - name: {{ $key }}
          value: "{{ $value }}"
        {{- end }}
`
	replicas := 1
	if r, ok := node.Config["replicas"].(float64); ok {
		replicas = int(r)
	}

	port := 8080
	if p, ok := node.Config["port"].(float64); ok {
		port = int(p)
	}

	data := struct {
		ID       string
		Image    string
		Replicas int
		Port     int
		Env      map[string]interface{}
	}{
		ID:       node.ID,
		Image:    image,
		Replicas: replicas,
		Port:     port,
		Env:      make(map[string]interface{}),
	}

	// Extract env vars if present
	if envs, ok := node.Config["environment_variables"].(map[string]interface{}); ok {
		data.Env = envs
	}

	t, _ := template.New("deploy").Parse(tmpl)
	var out bytes.Buffer
	t.Execute(&out, data)
	return out.String(), nil
}

func (g *HelmGenerator) generateService(node models.Node) (string, error) {
	port := 80
	targetPort := 8080
	if p, ok := node.Config["port"].(float64); ok {
		targetPort = int(p)
	}

	tmpl := `apiVersion: v1
kind: Service
metadata:
  name: {{ .ID }}
spec:
  selector:
    app: {{ .ID }}
  ports:
  - protocol: TCP
    port: {{ .Port }}
    targetPort: {{ .TargetPort }}
  type: ClusterIP
`
	data := struct {
		ID         string
		Port       int
		TargetPort int
	}{
		ID:         node.ID,
		Port:       port,
		TargetPort: targetPort,
	}

	t, _ := template.New("svc").Parse(tmpl)
	var out bytes.Buffer
	t.Execute(&out, data)
	return out.String(), nil
}

func (g *HelmGenerator) generateHookConfigMap(conn models.Connection, index int) (string, error) {
	tmpl := `apiVersion: v1
kind: ConfigMap
metadata:
  name: hook-code-{{ .Index }}
data:
  hook.go: |
{{ .HookCode | indent 4 }}
`
	funcMap := template.FuncMap{
		"indent": func(spaces int, v string) string {
			pad := ""
			for i := 0; i < spaces; i++ {
				pad += " "
			}
			return pad + v
		},
	}

	data := struct {
		Index    int
		HookCode string
	}{
		Index:    index,
		HookCode: conn.HookCode,
	}

	t, _ := template.New("cm").Funcs(funcMap).Parse(tmpl)
	var out bytes.Buffer
	t.Execute(&out, data)
	return out.String(), nil
}

func (g *HelmGenerator) generateHookProxy(conn models.Connection, index int) (string, error) {
	tmpl := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: hook-proxy-{{ .Index }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hook-proxy-{{ .Index }}
  template:
    metadata:
      labels:
        app: hook-proxy-{{ .Index }}
    spec:
      containers:
      - name: proxy
        image: infrastructure/hook-proxy:latest
        env:
        - name: TARGET_URL
          value: "http://{{ .ToID }}"
        volumeMounts:
        - name: code
          mountPath: /etc/hook
      volumes:
      - name: code
        configMap:
          name: hook-code-{{ .Index }}
---
apiVersion: v1
kind: Service
metadata:
  name: hook-proxy-{{ .Index }}
spec:
  selector:
    app: hook-proxy-{{ .Index }}
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
`
	data := struct {
		Index int
		ToID  string
	}{
		Index: index,
		ToID:  conn.ToNodeID,
	}

	t, _ := template.New("proxy").Parse(tmpl)
	var out bytes.Buffer
	t.Execute(&out, data)
	return out.String(), nil
}
