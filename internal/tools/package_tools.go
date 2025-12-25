package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dev-cli/internal/executor"
)

// PackageInfoTool analyzes project dependencies.
type PackageInfoTool struct{}

func (t *PackageInfoTool) Name() string        { return "package_info" }
func (t *PackageInfoTool) Description() string { return "Analyze project dependencies (Go, npm, pip)" }

func (t *PackageInfoTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "type", Type: "string", Description: "Package type: auto, go, npm, pip", Required: false, Default: "auto"},
		{Name: "action", Type: "string", Description: "Action: list, outdated, check", Required: false, Default: "list"},
		{Name: "path", Type: "string", Description: "Project path", Required: false, Default: "."},
	}
}

// PackageResult contains dependency analysis results.
type PackageResult struct {
	Type        string        `json:"type"`
	Path        string        `json:"path"`
	Packages    []PackageInfo `json:"packages,omitempty"`
	Outdated    []PackageInfo `json:"outdated,omitempty"`
	TotalCount  int           `json:"total_count"`
	DirectCount int           `json:"direct_count"`
}

// PackageInfo represents a single package/dependency.
type PackageInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Latest   string `json:"latest,omitempty"`
	Direct   bool   `json:"direct,omitempty"`
	Indirect bool   `json:"indirect,omitempty"`
}

func (t *PackageInfoTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	pkgType := GetString(params, "type", "auto")
	action := GetString(params, "action", "list")
	path := GetString(params, "path", ".")

	if pkgType == "auto" {
		pkgType = detectPackageType(path)
		if pkgType == "" {
			return NewErrorResult("could not detect package type, specify 'type' parameter", time.Since(start))
		}
	}

	switch pkgType {
	case "go":
		return t.analyzeGo(path, action, start)
	case "npm":
		return t.analyzeNpm(path, action, start)
	case "pip":
		return t.analyzePip(path, action, start)
	default:
		return NewErrorResult("unknown package type: "+pkgType, time.Since(start))
	}
}

func detectPackageType(path string) string {
	if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
		return "go"
	}
	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		return "npm"
	}
	if _, err := os.Stat(filepath.Join(path, "requirements.txt")); err == nil {
		return "pip"
	}
	if _, err := os.Stat(filepath.Join(path, "pyproject.toml")); err == nil {
		return "pip"
	}
	return ""
}

func (t *PackageInfoTool) analyzeGo(path, action string, start time.Time) ToolResult {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	switch action {
	case "list":

		result := executor.ExecuteSimple("cd " + absPath + " && go list -m -f '{{.Path}}@{{.Version}}' all 2>/dev/null | head -100")

		packages := make([]PackageInfo, 0)
		directCount := 0

		lines := strings.Split(result.Output, "\n")
		for i, line := range lines {
			line = strings.Trim(line, "'")
			if line == "" {
				continue
			}
			parts := strings.Split(line, "@")
			if len(parts) == 2 {
				pkg := PackageInfo{
					Name:    parts[0],
					Version: parts[1],
					Direct:  i == 0,
				}
				packages = append(packages, pkg)
				if pkg.Direct {
					directCount++
				}
			}
		}

		return NewResult(PackageResult{
			Type:        "go",
			Path:        absPath,
			Packages:    packages,
			TotalCount:  len(packages),
			DirectCount: directCount,
		}, time.Since(start))

	case "outdated":

		result := executor.ExecuteSimple("cd " + absPath + " && go list -u -m -f '{{if .Update}}{{.Path}}@{{.Version}}->{{.Update.Version}}{{end}}' all 2>/dev/null")

		outdated := make([]PackageInfo, 0)
		for _, line := range strings.Split(result.Output, "\n") {
			line = strings.Trim(line, "'")
			if line == "" {
				continue
			}

			parts := strings.Split(line, "->")
			if len(parts) == 2 {
				nameParts := strings.Split(parts[0], "@")
				if len(nameParts) == 2 {
					outdated = append(outdated, PackageInfo{
						Name:    nameParts[0],
						Version: nameParts[1],
						Latest:  parts[1],
					})
				}
			}
		}

		return NewResult(PackageResult{
			Type:       "go",
			Path:       absPath,
			Outdated:   outdated,
			TotalCount: len(outdated),
		}, time.Since(start))

	default:
		return NewErrorResult("unknown action for go: "+action, time.Since(start))
	}
}

func (t *PackageInfoTool) analyzeNpm(path, action string, start time.Time) ToolResult {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	switch action {
	case "list":

		pkgPath := filepath.Join(absPath, "package.json")
		data, err := os.ReadFile(pkgPath)
		if err != nil {
			return NewErrorResult("cannot read package.json: "+err.Error(), time.Since(start))
		}

		var pkg struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			return NewErrorResult("invalid package.json: "+err.Error(), time.Since(start))
		}

		packages := make([]PackageInfo, 0)
		for name, version := range pkg.Dependencies {
			packages = append(packages, PackageInfo{Name: name, Version: version, Direct: true})
		}
		for name, version := range pkg.DevDependencies {
			packages = append(packages, PackageInfo{Name: name, Version: version, Direct: true})
		}

		return NewResult(PackageResult{
			Type:        "npm",
			Path:        absPath,
			Packages:    packages,
			TotalCount:  len(packages),
			DirectCount: len(packages),
		}, time.Since(start))

	case "outdated":
		result := executor.ExecuteSimple("cd " + absPath + " && npm outdated --json 2>/dev/null")

		var outdatedMap map[string]struct {
			Current string `json:"current"`
			Latest  string `json:"latest"`
		}

		outdated := make([]PackageInfo, 0)
		if err := json.Unmarshal([]byte(result.Output), &outdatedMap); err == nil {
			for name, info := range outdatedMap {
				outdated = append(outdated, PackageInfo{
					Name:    name,
					Version: info.Current,
					Latest:  info.Latest,
				})
			}
		}

		return NewResult(PackageResult{
			Type:       "npm",
			Path:       absPath,
			Outdated:   outdated,
			TotalCount: len(outdated),
		}, time.Since(start))

	default:
		return NewErrorResult("unknown action for npm: "+action, time.Since(start))
	}
}

func (t *PackageInfoTool) analyzePip(path, action string, start time.Time) ToolResult {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	switch action {
	case "list":

		reqPath := filepath.Join(absPath, "requirements.txt")
		data, err := os.ReadFile(reqPath)
		if err != nil {

			result := executor.ExecuteSimple("pip list --format=json 2>/dev/null")
			return t.parsePipList(result.Output, absPath, start)
		}

		packages := make([]PackageInfo, 0)
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
				continue
			}

			for _, sep := range []string{"==", ">=", "<=", "~=", "!="} {
				if parts := strings.SplitN(line, sep, 2); len(parts) == 2 {
					packages = append(packages, PackageInfo{Name: parts[0], Version: parts[1], Direct: true})
					break
				}
			}
		}

		return NewResult(PackageResult{
			Type:        "pip",
			Path:        absPath,
			Packages:    packages,
			TotalCount:  len(packages),
			DirectCount: len(packages),
		}, time.Since(start))

	case "outdated":
		result := executor.ExecuteSimple("pip list --outdated --format=json 2>/dev/null")

		var outdatedList []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Latest  string `json:"latest_version"`
		}

		outdated := make([]PackageInfo, 0)
		if err := json.Unmarshal([]byte(result.Output), &outdatedList); err == nil {
			for _, pkg := range outdatedList {
				outdated = append(outdated, PackageInfo{
					Name:    pkg.Name,
					Version: pkg.Version,
					Latest:  pkg.Latest,
				})
			}
		}

		return NewResult(PackageResult{
			Type:       "pip",
			Path:       absPath,
			Outdated:   outdated,
			TotalCount: len(outdated),
		}, time.Since(start))

	default:
		return NewErrorResult("unknown action for pip: "+action, time.Since(start))
	}
}

func (t *PackageInfoTool) parsePipList(output, path string, start time.Time) ToolResult {
	var pkgList []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	packages := make([]PackageInfo, 0)
	if err := json.Unmarshal([]byte(output), &pkgList); err == nil {
		for _, pkg := range pkgList {
			packages = append(packages, PackageInfo{Name: pkg.Name, Version: pkg.Version})
		}
	}

	return NewResult(PackageResult{
		Type:       "pip",
		Path:       path,
		Packages:   packages,
		TotalCount: len(packages),
	}, time.Since(start))
}
