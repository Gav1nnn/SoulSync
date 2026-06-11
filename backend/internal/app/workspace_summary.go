package app

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxWorkspaceScanFiles = 700
	maxTreeItems          = 80
	maxCandidateItems     = 12
	maxProjectDocSnippets = 4
	maxProjectDocChars    = 1600
)

var skippedWorkspaceDirs = map[string]bool{
	".data":         true,
	".git":          true,
	".next":         true,
	".nuxt":         true,
	".turbo":        true,
	".venv":         true,
	"build":         true,
	"coverage":      true,
	"dist":          true,
	"node_modules":  true,
	"target":        true,
	"vendor":        true,
	"__pycache__":   true,
	".pytest_cache": true,
}

type scannedWorkspaceFile struct {
	absPath string
	relPath string
	info    fs.DirEntry
}

func buildWorkspaceSummary(root string) (WorkspaceSummary, error) {
	files, err := scanWorkspaceFiles(root)
	if err != nil {
		return WorkspaceSummary{}, err
	}

	projectDocCandidates := detectProjectDocCandidates(files)
	return WorkspaceSummary{
		WorkspacePath:           root,
		RootName:                filepath.Base(root),
		Tree:                    summarizeWorkspaceTree(files),
		PackageManagers:         detectPackageManagers(root, files),
		FrontendFrameworks:      detectFrontendFrameworks(root, files),
		BackendFrameworks:       detectBackendFrameworks(files),
		BackendRouteCandidates:  detectBackendRouteCandidates(files),
		APICandidates:           detectAPICandidates(files),
		ProjectDocCandidates:    projectDocCandidates,
		ProjectDocSnippets:      detectProjectDocSnippets(files, projectDocCandidates),
		TypeFileCandidates:      detectTypeFileCandidates(files),
		FrontendEntryCandidates: detectFrontendEntryCandidates(files),
		APIClientCandidates:     detectAPIClientCandidates(files),
		ValidationCommands:      detectValidationCommands(root, files),
		GeneratedAt:             time.Now(),
	}, nil
}

func scanWorkspaceFiles(root string) ([]scannedWorkspaceFile, error) {
	files := make([]scannedWorkspaceFile, 0, 128)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		if entry.IsDir() && shouldSkipWorkspaceDir(entry.Name()) {
			return filepath.SkipDir
		}
		if len(files) >= maxWorkspaceScanFiles {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		files = append(files, scannedWorkspaceFile{
			absPath: path,
			relPath: relPath,
			info:    entry,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan workspace: %w", err)
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].relPath < files[j].relPath
	})

	return files, nil
}

func summarizeWorkspaceTree(files []scannedWorkspaceFile) []WorkspaceTreeItem {
	tree := make([]WorkspaceTreeItem, 0, maxTreeItems)
	for _, file := range files {
		if len(tree) >= maxTreeItems {
			break
		}
		if workspacePathDepth(file.relPath) > 2 {
			continue
		}

		itemType := "file"
		if file.info.IsDir() {
			itemType = "dir"
		}
		tree = append(tree, WorkspaceTreeItem{
			Path: file.relPath,
			Type: itemType,
		})
	}

	return tree
}

func detectPackageManagers(root string, files []scannedWorkspaceFile) []string {
	found := map[string]bool{}
	addIfExists := func(filename string, manager string) {
		if hasWorkspaceFile(files, filename) {
			found[manager] = true
		}
	}

	addIfExists("package-lock.json", "npm")
	addIfExists("pnpm-lock.yaml", "pnpm")
	addIfExists("yarn.lock", "yarn")
	addIfExists("bun.lockb", "bun")
	addIfExists("uv.lock", "uv")
	addIfExists("go.mod", "go")
	if hasWorkspaceFile(files, "requirements.txt") || hasWorkspaceFile(files, "pyproject.toml") {
		found["python"] = true
	}
	if hasWorkspaceFile(files, "package.json") && !found["npm"] && !found["pnpm"] && !found["yarn"] && !found["bun"] {
		found["npm"] = true
	}

	_ = root
	return sortedKeys(found)
}

func detectFrontendFrameworks(root string, files []scannedWorkspaceFile) []string {
	found := map[string]bool{}
	for _, file := range files {
		if filepath.Base(file.relPath) != "package.json" {
			continue
		}

		dependencies, err := readPackageDependencies(file.absPath)
		if err != nil {
			continue
		}
		for dependency := range dependencies {
			switch dependency {
			case "vue":
				found["Vue"] = true
			case "react":
				found["React"] = true
			case "svelte":
				found["Svelte"] = true
			case "next":
				found["Next.js"] = true
			case "vite":
				found["Vite"] = true
			}
		}
	}

	if len(found) == 0 {
		for _, file := range files {
			switch {
			case strings.HasSuffix(file.relPath, ".vue"):
				found["Vue"] = true
			case strings.HasSuffix(file.relPath, ".tsx") || strings.HasSuffix(file.relPath, ".jsx"):
				found["React"] = true
			}
		}
	}

	_ = root
	return sortedKeys(found)
}

func detectBackendFrameworks(files []scannedWorkspaceFile) []string {
	found := map[string]bool{}
	for _, file := range files {
		if !isTextCodeFile(file.relPath) {
			continue
		}
		content := readSmallText(file.absPath)
		switch {
		case strings.Contains(content, "github.com/gin-gonic/gin"):
			found["Gin"] = true
		case strings.Contains(content, "fastapi") || strings.Contains(content, "FastAPI"):
			found["FastAPI"] = true
		case strings.Contains(content, "github.com/gofiber/fiber"):
			found["Fiber"] = true
		case strings.Contains(content, "express"):
			found["Express"] = true
		}
	}

	return sortedKeys(found)
}

func detectBackendRouteCandidates(files []scannedWorkspaceFile) []WorkspaceCandidate {
	candidates := make([]WorkspaceCandidate, 0, maxCandidateItems)
	for _, file := range files {
		if len(candidates) >= maxCandidateItems {
			break
		}
		if !isTextCodeFile(file.relPath) {
			continue
		}
		content := readSmallText(file.absPath)
		switch {
		case strings.HasSuffix(file.relPath, ".go") && looksLikeGinRouteFile(content):
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "go.gin.routes", Reason: "contains Gin route registration"})
		case strings.HasSuffix(file.relPath, ".py") && looksLikeFastAPIRouteFile(content):
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "python.fastapi.routes", Reason: "contains FastAPI route decorators"})
		case strings.HasSuffix(file.relPath, ".ts") && strings.Contains(content, "Router("):
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "node.routes", Reason: "contains router construction"})
		}
	}

	return candidates
}

func detectAPICandidates(files []scannedWorkspaceFile) []APICandidate {
	candidates := make([]APICandidate, 0, maxCandidateItems)
	for _, file := range files {
		if len(candidates) >= maxCandidateItems {
			break
		}
		if file.info.IsDir() || !strings.HasSuffix(file.relPath, ".go") {
			continue
		}
		content := readSmallText(file.absPath)
		if !looksLikeGinRouteFile(content) {
			continue
		}
		fileCandidates := detectGinAPICandidates(file, content)
		for _, candidate := range fileCandidates {
			candidates = append(candidates, candidate)
			if len(candidates) >= maxCandidateItems {
				break
			}
		}
	}

	return candidates
}

func detectProjectDocCandidates(files []scannedWorkspaceFile) []WorkspaceCandidate {
	candidates := make([]WorkspaceCandidate, 0, maxCandidateItems)
	for _, file := range files {
		if len(candidates) >= maxCandidateItems {
			break
		}
		if file.info.IsDir() {
			continue
		}

		lowerPath := strings.ToLower(file.relPath)
		base := strings.ToLower(filepath.Base(file.relPath))
		switch {
		case strings.HasPrefix(base, "readme"):
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "project.readme", Reason: "README candidate for project conventions"})
		case strings.HasPrefix(lowerPath, "docs/") || strings.Contains(lowerPath, "/docs/"):
			if isProjectDocFile(lowerPath) {
				candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "project.docs", Reason: "project documentation candidate"})
			}
		case strings.Contains(lowerPath, "openapi") || strings.Contains(lowerPath, "swagger"):
			if isProjectDocFile(lowerPath) {
				candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "project.api_docs", Reason: "API documentation candidate"})
			}
		case strings.Contains(lowerPath, "api") && strings.Contains(lowerPath, "doc"):
			if isProjectDocFile(lowerPath) {
				candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "project.api_docs", Reason: "API documentation candidate"})
			}
		}
	}

	return candidates
}

func detectProjectDocSnippets(files []scannedWorkspaceFile, candidates []WorkspaceCandidate) []ProjectDocSnippet {
	if len(candidates) == 0 {
		return []ProjectDocSnippet{}
	}

	fileByPath := map[string]scannedWorkspaceFile{}
	for _, file := range files {
		fileByPath[file.relPath] = file
	}

	snippets := make([]ProjectDocSnippet, 0, min(maxProjectDocSnippets, len(candidates)))
	for _, candidate := range candidates {
		if len(snippets) >= maxProjectDocSnippets {
			break
		}
		file, ok := fileByPath[candidate.Path]
		if !ok {
			continue
		}
		content := normalizeProjectDocSnippet(readSmallText(file.absPath))
		if content == "" {
			continue
		}
		snippets = append(snippets, ProjectDocSnippet{
			Path:    candidate.Path,
			Kind:    candidate.Kind,
			Content: content,
		})
	}

	return snippets
}

func normalizeProjectDocSnippet(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	content = strings.Join(strings.Fields(content), " ")
	runes := []rune(content)
	if len(runes) > maxProjectDocChars {
		return string(runes[:maxProjectDocChars]) + " ... truncated"
	}
	return content
}

func detectGinAPICandidates(file scannedWorkspaceFile, content string) []APICandidate {
	fileSet := token.NewFileSet()
	parsedFile, err := parser.ParseFile(fileSet, file.absPath, content, 0)
	if err != nil {
		return []APICandidate{}
	}

	groupPrefixes := map[string]string{}
	structCandidates := ginStructCandidates(file.relPath, parsedFile)
	candidates := []APICandidate{}
	ast.Inspect(parsedFile, func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if ok {
			recordGinGroupAssignment(groupPrefixes, assign)
			return true
		}

		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		method := selector.Sel.Name
		if !isGinHTTPMethod(method) {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		routePath, ok := stringLiteralValue(call.Args[0])
		if !ok {
			return true
		}
		receiver := exprIdentName(selector.X)
		fullPath := joinAPIPath(groupPrefixes[receiver], routePath)
		candidates = append(candidates, APICandidate{
			Method:          method,
			Path:            fullPath,
			Handler:         handlerName(call.Args[1]),
			HandlerFile:     file.relPath,
			TypeDefinitions: structCandidates,
			Reason:          "detected Gin route registration",
		})
		return true
	})

	return candidates
}

func recordGinGroupAssignment(groupPrefixes map[string]string, assign *ast.AssignStmt) {
	if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
		return
	}
	target, ok := assign.Lhs[0].(*ast.Ident)
	if !ok {
		return
	}
	call, ok := assign.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Group" || len(call.Args) == 0 {
		return
	}
	groupPath, ok := stringLiteralValue(call.Args[0])
	if !ok {
		return
	}
	parent := exprIdentName(selector.X)
	groupPrefixes[target.Name] = joinAPIPath(groupPrefixes[parent], groupPath)
}

func ginStructCandidates(path string, parsedFile *ast.File) []WorkspaceCandidate {
	candidates := []WorkspaceCandidate{}
	for _, declaration := range parsedFile.Decls {
		generalDeclaration, ok := declaration.(*ast.GenDecl)
		if !ok || generalDeclaration.Tok != token.TYPE {
			continue
		}
		for _, spec := range generalDeclaration.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := typeSpec.Type.(*ast.StructType); !ok {
				continue
			}
			candidates = append(candidates, WorkspaceCandidate{
				Path:   path,
				Kind:   "go.struct",
				Reason: "struct " + typeSpec.Name.Name + " is defined near detected Gin routes",
			})
			if len(candidates) >= 6 {
				return candidates
			}
		}
	}

	return candidates
}

func isGinHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func stringLiteralValue(expression ast.Expr) (string, bool) {
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value := strings.Trim(literal.Value, "`\"")
	if value == "" {
		value = "/"
	}
	return value, true
}

func exprIdentName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		return value.Sel.Name
	default:
		return ""
	}
}

func handlerName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		if receiver := exprIdentName(value.X); receiver != "" {
			return receiver + "." + value.Sel.Name
		}
		return value.Sel.Name
	case *ast.CallExpr:
		return handlerName(value.Fun)
	case *ast.FuncLit:
		return "inline handler"
	default:
		return ""
	}
}

func joinAPIPath(prefix string, path string) string {
	trimmedPrefix := strings.TrimSuffix(strings.TrimSpace(prefix), "/")
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" || trimmedPath == "/" {
		if trimmedPrefix == "" {
			return "/"
		}
		return trimmedPrefix
	}
	if !strings.HasPrefix(trimmedPath, "/") {
		trimmedPath = "/" + trimmedPath
	}
	if trimmedPrefix == "" {
		return trimmedPath
	}
	return trimmedPrefix + trimmedPath
}

func detectTypeFileCandidates(files []scannedWorkspaceFile) []WorkspaceCandidate {
	candidates := make([]WorkspaceCandidate, 0, maxCandidateItems)
	for _, file := range files {
		if len(candidates) >= maxCandidateItems {
			break
		}
		lowerPath := strings.ToLower(file.relPath)
		base := strings.ToLower(filepath.Base(file.relPath))
		if file.info.IsDir() {
			continue
		}
		if strings.HasSuffix(lowerPath, ".d.ts") ||
			strings.Contains(lowerPath, "/types/") ||
			strings.Contains(lowerPath, "/schemas/") ||
			strings.Contains(lowerPath, "/dto/") ||
			strings.Contains(base, "types") ||
			strings.Contains(base, "schema") ||
			strings.Contains(base, "dto") {
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "types", Reason: "path suggests shared type or schema definitions"})
		}
	}

	return candidates
}

func detectFrontendEntryCandidates(files []scannedWorkspaceFile) []WorkspaceCandidate {
	candidates := make([]WorkspaceCandidate, 0, maxCandidateItems)
	for _, file := range files {
		if len(candidates) >= maxCandidateItems {
			break
		}
		lowerPath := strings.ToLower(file.relPath)
		if file.info.IsDir() {
			continue
		}
		switch {
		case strings.Contains(lowerPath, "/views/") || strings.Contains(lowerPath, "/pages/"):
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "frontend.page", Reason: "path is under views/pages"})
		case strings.HasSuffix(lowerPath, "router.ts") || strings.HasSuffix(lowerPath, "router.js"):
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "frontend.router", Reason: "router entry candidate"})
		case strings.HasSuffix(lowerPath, "main.ts") || strings.HasSuffix(lowerPath, "main.js") || strings.HasSuffix(lowerPath, "app.vue"):
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "frontend.entry", Reason: "frontend app entry candidate"})
		}
	}

	return candidates
}

func detectAPIClientCandidates(files []scannedWorkspaceFile) []WorkspaceCandidate {
	candidates := make([]WorkspaceCandidate, 0, maxCandidateItems)
	for _, file := range files {
		if len(candidates) >= maxCandidateItems {
			break
		}
		if file.info.IsDir() {
			continue
		}
		lowerPath := strings.ToLower(file.relPath)
		if strings.Contains(lowerPath, "/api/") ||
			strings.Contains(lowerPath, "/client/") ||
			strings.Contains(lowerPath, "request") ||
			strings.Contains(lowerPath, "http") ||
			strings.Contains(lowerPath, "axios") {
			candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "frontend.api_client", Reason: "path suggests API client or request wrapper"})
			continue
		}

		if isTextCodeFile(file.relPath) {
			content := readSmallText(file.absPath)
			if strings.Contains(content, "fetch(") || strings.Contains(content, "axios.") {
				candidates = append(candidates, WorkspaceCandidate{Path: file.relPath, Kind: "frontend.api_client", Reason: "contains fetch or axios usage"})
			}
		}
	}

	return candidates
}

func detectValidationCommands(root string, files []scannedWorkspaceFile) []string {
	commands := make([]string, 0, 8)
	for _, file := range files {
		if filepath.Base(file.relPath) != "package.json" {
			continue
		}
		for _, script := range readPackageScripts(file.absPath) {
			if script == "build" || script == "test" || script == "lint" || script == "typecheck" {
				commands = append(commands, commandForPackageScript(file.relPath, script))
			}
		}
	}
	if hasWorkspaceFile(files, "go.mod") {
		commands = append(commands, "go test ./...")
	}
	if hasWorkspaceFile(files, "uv.lock") || hasWorkspaceFile(files, "pyproject.toml") {
		commands = append(commands, "uv run python -m unittest discover -s tests -p 'test_*.py'")
	} else if hasWorkspaceFile(files, "requirements.txt") {
		commands = append(commands, "python -m unittest discover -s tests -p 'test_*.py'")
	}

	_ = root
	return uniqueStrings(commands)
}

func readPackageDependencies(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var packageJSON struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(packageJSON.Dependencies)+len(packageJSON.DevDependencies))
	for key, value := range packageJSON.Dependencies {
		result[key] = value
	}
	for key, value := range packageJSON.DevDependencies {
		result[key] = value
	}

	return result, nil
}

func readPackageScripts(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{}
	}

	var packageJSON struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return []string{}
	}

	scripts := make([]string, 0, len(packageJSON.Scripts))
	for script := range packageJSON.Scripts {
		scripts = append(scripts, script)
	}
	sort.Strings(scripts)
	return scripts
}

func commandForPackageScript(packagePath string, script string) string {
	dir := filepath.ToSlash(filepath.Dir(packagePath))
	if dir == "." {
		return "npm run " + script
	}
	return "cd " + dir + " && npm run " + script
}

func readSmallText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if len(data) > 64*1024 {
		data = data[:64*1024]
	}
	return string(data)
}

func looksLikeGinRouteFile(content string) bool {
	methods := []string{".GET(", ".POST(", ".PUT(", ".PATCH(", ".DELETE(", ".Group("}
	for _, method := range methods {
		if strings.Contains(content, method) {
			return true
		}
	}
	return false
}

func looksLikeFastAPIRouteFile(content string) bool {
	methods := []string{"@app.get", "@app.post", "@app.put", "@app.patch", "@app.delete", "@router.get", "@router.post"}
	for _, method := range methods {
		if strings.Contains(content, method) {
			return true
		}
	}
	return false
}

func hasWorkspaceFile(files []scannedWorkspaceFile, relPath string) bool {
	for _, file := range files {
		if file.relPath == relPath || strings.HasSuffix(file.relPath, "/"+relPath) {
			return true
		}
	}
	return false
}

func shouldSkipWorkspaceDir(name string) bool {
	return skippedWorkspaceDirs[name]
}

func workspacePathDepth(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/") + 1
}

func isTextCodeFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".py", ".js", ".jsx", ".ts", ".tsx", ".vue":
		return true
	default:
		return false
	}
}

func isProjectDocFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".mdx", ".txt", ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
