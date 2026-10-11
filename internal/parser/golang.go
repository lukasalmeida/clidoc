package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lukasalmeida/clidoc/internal/ir"
)

// ParseDirectory parses all Go source files under root and extracts routes from comment annotations.
func ParseDirectory(root string) ([]ir.Route, error) {
	if root == "" {
		root = "."
	}

	var routes []ir.Route
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".idea" || d.Name() == ".vscode" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		fileRoutes, err := parseFile(path)
		if err != nil {
			return err
		}
		routes = append(routes, fileRoutes...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return routes, nil
}

func parseFile(path string) ([]ir.Route, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var routes []ir.Route
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Doc == nil {
			continue
		}
		route, ok := parseRouteAnnotations(fn.Doc)
		if !ok {
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

func parseRouteAnnotations(doc *ast.CommentGroup) (ir.Route, bool) {
	var route ir.Route
	for _, comment := range doc.List {
		line := strings.TrimSpace(comment.Text)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "@Route"):
			method, path, ok := parseRouteLine(line)
			if !ok {
				continue
			}
			route.Method = strings.ToUpper(method)
			route.Path = path
		case strings.HasPrefix(line, "@Summary"):
			route.Summary = strings.TrimSpace(strings.TrimPrefix(line, "@Summary"))
		case strings.HasPrefix(line, "@Description"):
			route.Description = strings.TrimSpace(strings.TrimPrefix(line, "@Description"))
		case strings.HasPrefix(line, "@Tag"):
			route.Tag = strings.TrimSpace(strings.TrimPrefix(line, "@Tag"))
		case strings.HasPrefix(line, "@Param"):
			param, ok := parseParamLine(line)
			if ok {
				route.Params = append(route.Params, param)
			}
		case strings.HasPrefix(line, "@Success"):
			resp, ok := parseResponseLine(line)
			if ok {
				route.Responses = append(route.Responses, resp)
			}
		case strings.HasPrefix(line, "@Failure"):
			resp, ok := parseResponseLine(line)
			if ok {
				route.Responses = append(route.Responses, resp)
			}
		}
	}

	if route.Method == "" || route.Path == "" {
		return ir.Route{}, false
	}

	return route, true
}

func parseRouteLine(line string) (string, string, bool) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(line, "@Route"))
	parts := strings.Fields(trimmed)
	if len(parts) < 2 {
		return "", "", false
	}
	method := strings.ToUpper(parts[0])
	path := parts[1]
	return method, path, true
}

var paramPattern = regexp.MustCompile(`^@Param\s+(\S+)\s+(\S+)\s+(\S+)\s+(true|false)\s+"?(.*?)(?:"?)$`)

func parseParamLine(line string) (ir.Parameter, bool) {
	matches := paramPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 6 {
		return ir.Parameter{}, false
	}

	required, err := strconv.ParseBool(matches[4])
	if err != nil {
		return ir.Parameter{}, false
	}

	return ir.Parameter{
		Name:        matches[1],
		In:          matches[2],
		Type:        matches[3],
		Required:    required,
		Description: strings.Trim(matches[5], `"`),
	}, true
}

var responsePattern = regexp.MustCompile(`^@(?:Success|Failure)\s+(\d+)\s+(\S+)\s+"?(.*?)(?:"?)$`)

func parseResponseLine(line string) (ir.Response, bool) {
	matches := responsePattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 4 {
		return ir.Response{}, false
	}

	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return ir.Response{}, false
	}

	return ir.Response{
		StatusCode:  code,
		Type:        matches[2],
		Description: strings.Trim(matches[3], `"`),
	}, true
}
