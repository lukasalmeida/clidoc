package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lukasalmeida/clidoc/internal/ir"
)

type OpenAPISpec struct {
	OpenAPI string                          `json:"openapi"`
	Info    Info                            `json:"info"`
	Servers []Server                        `json:"servers,omitempty"`
	Paths   map[string]map[string]Operation `json:"paths"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Server struct {
	URL string `json:"url"`
}

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Schema      Schema `json:"schema"`
}

type Schema struct {
	Type string `json:"type,omitempty"`
}

type Response struct {
	Description string             `json:"description"`
	Content     map[string]Content `json:"content,omitempty"`
}

type Content struct {
	Schema Schema `json:"schema"`
}

func GenerateSpec(api ir.API) OpenAPISpec {
	if api.Title == "" {
		api.Title = "CliDoc API"
	}
	if api.Version == "" {
		api.Version = "1.0.0"
	}

	paths := make(map[string]map[string]Operation)
	for _, route := range api.Endpoints {
		method := strings.ToLower(route.Method)
		if method == "" {
			continue
		}
		if _, ok := paths[route.Path]; !ok {
			paths[route.Path] = make(map[string]Operation)
		}

		paths[route.Path][method] = Operation{
			Summary:     route.Summary,
			Description: route.Description,
			Tags:        tagList(route.Tag),
			Parameters:  buildParameters(route.Params),
			Responses:   buildResponses(route.Responses),
		}
	}

	return OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:   api.Title,
			Version: api.Version,
		},
		Servers: []Server{{URL: api.BaseURL}},
		Paths:   paths,
	}
}

func buildParameters(params []ir.Parameter) []Parameter {
	if len(params) == 0 {
		return nil
	}

	result := make([]Parameter, 0, len(params))
	for _, p := range params {
		result = append(result, Parameter{
			Name:        p.Name,
			In:          p.In,
			Required:    p.Required,
			Description: p.Description,
			Schema: Schema{
				Type: schemaType(p.Type),
			},
		})
	}
	return result
}

func buildResponses(responses []ir.Response) map[string]Response {
	if len(responses) == 0 {
		return map[string]Response{
			"200": {
				Description: "OK",
			},
		}
	}

	result := make(map[string]Response, len(responses))
	for _, r := range responses {
		contentType := "application/json"
		if strings.EqualFold(r.Type, "text") || strings.EqualFold(r.Type, "plain") {
			contentType = "text/plain"
		}
		result[fmt.Sprintf("%d", r.StatusCode)] = Response{
			Description: r.Description,
			Content: map[string]Content{
				contentType: {
					Schema: Schema{Type: schemaType(r.Type)},
				},
			},
		}
	}
	return result
}

func tagList(tag string) []string {
	if strings.TrimSpace(tag) == "" {
		return nil
	}
	return []string{tag}
}

func schemaType(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "bool", "boolean":
		return "boolean"
	case "int", "int32", "int64", "integer":
		return "integer"
	case "float", "double", "number", "decimal":
		return "number"
	case "json", "object":
		return "object"
	case "", "any":
		return "string"
	default:
		return "string"
	}
}

func WriteFile(path string, api ir.API) error {
	spec := GenerateSpec(api)
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal openapi: %w", err)
	}
	if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write openapi: %w", err)
	}
	return nil
}

func filepathDir(path string) string {
	if idx := strings.LastIndex(path, "/"); idx > -1 {
		return path[:idx]
	}
	return "."
}

func SortPaths(spec *OpenAPISpec) {
	if spec == nil || spec.Paths == nil {
		return
	}
	keys := make([]string, 0, len(spec.Paths))
	for k := range spec.Paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]map[string]Operation, len(spec.Paths))
	for _, k := range keys {
		ordered[k] = spec.Paths[k]
	}
	spec.Paths = ordered
}
