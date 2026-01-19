package projects

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getarcaneapp/arcane/backend/internal/common"
	"github.com/getarcaneapp/arcane/types/project"
	"github.com/goccy/go-yaml"
)

var reservedRootFileNames = []string{
	"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml",
	".env",
}

const (
	// PlaceholderYAML is the placeholder content for new YAML files
	PlaceholderYAML = "# This file will be created when you save changes\nservices:\n"
	// PlaceholderGeneric is the placeholder content for new generic files
	PlaceholderGeneric = "# This file will be created when you save changes\n"
)

// ParseAllowedPaths parses a comma-separated string of allowed paths.
func ParseAllowedPaths(s string) []string {
	if s == "" {
		return nil
	}
	var paths []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" && filepath.IsAbs(t) {
			paths = append(paths, filepath.Clean(t))
		}
	}
	return paths
}

func resolveAbsPath(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	absDir = filepath.Clean(absDir)
	if evalDir, err := filepath.EvalSymlinks(absDir); err == nil {
		return evalDir, nil
	}
	return absDir, nil
}

func resolveFilePath(basePath, filePath string) (absPath, evalPath string, err error) {
	absPath = filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(basePath, filePath)
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", "", fmt.Errorf("invalid file path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		return absPath, resolved, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// File doesn't exist yet - evaluate parent directory symlinks
	dir := filepath.Dir(absPath)
	if evalDir, err := filepath.EvalSymlinks(dir); err == nil {
		return absPath, filepath.Join(evalDir, filepath.Base(absPath)), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("failed to resolve parent directory: %w", err)
	}
	return absPath, absPath, nil
}

func isWithinDirectory(evalPath, dir string) bool {
	if evalPath == dir {
		return false
	}
	evalPath = filepath.Clean(evalPath)
	prefix := dir + string(filepath.Separator)
	return strings.HasPrefix(evalPath+string(filepath.Separator), prefix)
}

func isWithinAllowedPaths(evalPath string, allowedPaths []string) bool {
	for _, ap := range allowedPaths {
		evalAllowedPath := ap
		if resolved, err := filepath.EvalSymlinks(ap); err == nil {
			evalAllowedPath = resolved
		}
		if isWithinDirectory(evalPath, evalAllowedPath) || evalPath == evalAllowedPath {
			return true
		}
	}
	return false
}

func isReservedFileName(absPath, absProjectDir string) bool {
	rel, err := filepath.Rel(absProjectDir, absPath)
	if err != nil || filepath.Dir(rel) != "." {
		return false
	}
	baseName := filepath.Base(rel)
	for _, r := range reservedRootFileNames {
		if strings.EqualFold(baseName, r) {
			return true
		}
	}
	return false
}

func readFileWithPlaceholder(absPath, placeholder string) (string, error) {
	content, err := os.ReadFile(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return placeholder, nil
		}
		return "", err
	}
	return string(content), nil
}

func writeFileWithDir(absPath, content string) error {
	if dir := filepath.Dir(absPath); dir != "" {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dir, common.DirPerm); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		}
	}
	return os.WriteFile(absPath, []byte(content), common.FilePerm)
}

// validatePath validates a file path for write operations.
// Returns the validated absolute path or an error.
func validatePath(projectDir, filePath string, allowedPaths []string, checkReserved bool) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	absProjectDir, err := resolveAbsPath(projectDir)
	if err != nil {
		return "", fmt.Errorf("invalid project directory: %w", err)
	}

	absPath, evalPath, err := resolveFilePath(absProjectDir, filePath)
	if err != nil {
		return "", err
	}

	if evalPath == absProjectDir {
		return "", fmt.Errorf("path cannot be the project directory itself")
	}

	withinProject := isWithinDirectory(evalPath, absProjectDir)
	withinAllowed := isWithinAllowedPaths(evalPath, allowedPaths)

	if !withinProject && !withinAllowed {
		if len(allowedPaths) == 0 {
			return "", fmt.Errorf("path outside project; configure ALLOWED_EXTERNAL_PATHS to allow external paths")
		}
		return "", fmt.Errorf("path not in project or allowed directories")
	}

	if checkReserved && withinProject && isReservedFileName(absPath, absProjectDir) {
		return "", fmt.Errorf("reserved file name: %s", filepath.Base(absPath))
	}

	return absPath, nil
}

// Security Model for Include Files:
// - READ: Docker Compose allows include files from anywhere (parent dirs, absolute paths, etc.)
//         We allow reading from any path to maintain compatibility with standard Docker Compose behavior
// - WRITE: Restricted to files within the project directory or configured allowed external paths

// ParseIncludes reads a compose file and extracts all include directives.
func ParseIncludes(composeFilePath string) ([]project.IncludeFile, error) {
	content, err := os.ReadFile(composeFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var composeData map[string]interface{}
	if err := yaml.Unmarshal(content, &composeData); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	includes, ok := composeData["include"]
	if !ok {
		return nil, nil
	}

	composeDir := filepath.Dir(composeFilePath)
	var files []project.IncludeFile

	parseItem := func(item interface{}) {
		inc, err := parseIncludeItem(item, composeDir)
		if err == nil {
			files = append(files, inc)
		}
	}

	switch v := includes.(type) {
	case []interface{}:
		for _, item := range v {
			parseItem(item)
		}
	case string:
		parseItem(v)
	}

	return files, nil
}

func parseIncludeItem(item interface{}, baseDir string) (project.IncludeFile, error) {
	var includePath string

	switch v := item.(type) {
	case string:
		includePath = v
	case map[string]interface{}:
		if path, ok := v["path"].(string); ok {
			includePath = path
		}
	default:
		return project.IncludeFile{}, fmt.Errorf("invalid include item type")
	}

	if includePath == "" {
		return project.IncludeFile{}, fmt.Errorf("empty include path")
	}

	fullPath := includePath
	if !filepath.IsAbs(includePath) {
		fullPath = filepath.Join(baseDir, includePath)
	}
	fullPath = filepath.Clean(fullPath)

	content, err := readFileWithPlaceholder(fullPath, PlaceholderYAML)
	if err != nil {
		return project.IncludeFile{}, fmt.Errorf("failed to read include file %s: %w", includePath, err)
	}

	relativePath := includePath
	if filepath.IsAbs(includePath) {
		if rel, err := filepath.Rel(baseDir, fullPath); err == nil {
			relativePath = rel
		}
	}

	return project.IncludeFile{
		Path:         fullPath,
		RelativePath: relativePath,
		Content:      content,
	}, nil
}

// WriteIncludeFile writes content to an include file path.
func WriteIncludeFile(projectDir, includePath, content string, allowedPaths []string) error {
	absPath, err := validatePath(projectDir, includePath, allowedPaths, false)
	if err != nil {
		return err
	}
	return writeFileWithDir(absPath, content)
}

// normalizePath returns a normalized path for storage (relative if within project, absolute otherwise).
func normalizePath(absPath, absProjectDir string) string {
	if rel, err := filepath.Rel(absProjectDir, absPath); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return absPath
}

// ValidateAndNormalizePath validates a file path and returns the normalized path for storage.
// The normalized path is relative if within the project directory, absolute otherwise.
func ValidateAndNormalizePath(projectDir, filePath string, allowedPaths []string, checkReserved bool) (string, error) {
	absPath, err := validatePath(projectDir, filePath, allowedPaths, checkReserved)
	if err != nil {
		return "", err
	}

	absProjectDir, _ := resolveAbsPath(projectDir)
	return normalizePath(absPath, absProjectDir), nil
}

// ReadCustomFileContents reads the contents of custom files given their paths from the database.
// Security: Validates paths against allowed external paths.
func ReadCustomFileContents(projectDir string, filePaths []string, allowedPaths []string) ([]project.CustomFile, error) {
	var files []project.CustomFile
	for _, path := range filePaths {
		absPath, err := validatePath(projectDir, path, allowedPaths, false)
		if err != nil {
			continue // Skip invalid paths
		}

		content, err := readFileWithPlaceholder(absPath, PlaceholderGeneric)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		files = append(files, project.CustomFile{
			Path:    path,
			Content: content,
		})
	}
	return files, nil
}

// WriteCustomFile writes content to a custom file.
func WriteCustomFile(projectDir, filePath, content string, allowedPaths []string) error {
	absPath, err := validatePath(projectDir, filePath, allowedPaths, true)
	if err != nil {
		return err
	}
	if err := writeFileWithDir(absPath, content); err != nil {
		return fmt.Errorf("failed to write custom file: %w", err)
	}
	return nil
}

// DeleteCustomFile deletes a custom file from disk.
func DeleteCustomFile(projectDir, filePath string, allowedPaths []string) error {
	absPath, err := validatePath(projectDir, filePath, allowedPaths, false)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file from disk: %w", err)
	}
	return nil
}
