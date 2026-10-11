package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lukasalmeida/clidoc/internal/generator"
	"github.com/lukasalmeida/clidoc/internal/ir"
)

func TestParseDirectory(t *testing.T) {
	dir := t.TempDir()

	content := `package sample

// @Route GET /api/v1/users
// @Summary Listar usuários
// @Description Retorna a lista de usuários.
// @Tag Usuários
// @Success 200 json "Lista retornada com sucesso"
func ListUsers() {}
`
	if err := os.WriteFile(filepath.Join(dir, "example.go"), []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	routes, err := ParseDirectory(dir)
	if err != nil {
		t.Fatalf("ParseDirectory returned error: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Method != "GET" {
		t.Fatalf("expected GET method, got %q", routes[0].Method)
	}
	if routes[0].Path != "/api/v1/users" {
		t.Fatalf("expected path /api/v1/users, got %q", routes[0].Path)
	}
}

func TestGenerateSpec(t *testing.T) {
	api := ir.API{
		Title:   "Sample API",
		Version: "1.0.0",
		BaseURL: "http://localhost:8080",
		Endpoints: []ir.Route{{
			Method:      "GET",
			Path:        "/api/v1/users",
			Summary:     "Listar usuários",
			Description: "Retorna a lista de usuários.",
			Tag:         "Usuários",
			Responses: []ir.Response{{
				StatusCode:  200,
				Type:        "json",
				Description: "Lista retornada com sucesso",
			}},
		}},
	}

	spec := generator.GenerateSpec(api)
	if spec.OpenAPI != "3.0.3" {
		t.Fatalf("expected openapi 3.0.3, got %q", spec.OpenAPI)
	}
	if _, ok := spec.Paths["/api/v1/users"]["get"]; !ok {
		t.Fatalf("expected generated GET route under /api/v1/users")
	}
}
