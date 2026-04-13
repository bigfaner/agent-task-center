// Package parser implements file format parsing for index.json, proposal.md, and manifest.md.
package parser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"agent-task-center/server/internal/model"
)

// ErrInvalidFile indicates the uploaded file has an unsupported format or invalid content.
var ErrInvalidFile = errors.New("invalid file format")

// Parser defines the interface for parsing task-related file formats.
type Parser interface {
	// ParseIndexJSON parses an index.json file and returns a list of TaskInput.
	ParseIndexJSON(data []byte) ([]model.TaskInput, error)
	// ParseProposalMD parses a proposal.md file, extracting the first H1 as title.
	ParseProposalMD(slug string, data []byte) (*model.ProposalInput, error)
	// ParseManifestMD parses a manifest.md file, extracting YAML frontmatter fields.
	ParseManifestMD(data []byte) (*model.FeatureInput, error)
}

// parser is the concrete implementation of Parser.
type parser struct{}

// New creates a new Parser instance.
func New() Parser {
	return &parser{}
}

// ---------------------------------------------------------------------------
// ParseIndexJSON
// ---------------------------------------------------------------------------

// indexFile represents the top-level structure of an index.json file.
type indexFile struct {
	Tasks map[string]taskEntry `json:"tasks"`
}

// taskEntry represents a single task entry within index.json.
// The JSON field "id" is mapped to TaskID (the actual index.json uses "id").
type taskEntry struct {
	ID           string   `json:"id"`
	TaskID       string   `json:"task_id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Priority     string   `json:"priority"`
	Tags         []string `json:"tags"`
	Dependencies []string `json:"dependencies"`
}

func (p *parser) ParseIndexJSON(data []byte) ([]model.TaskInput, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data: %w", ErrInvalidFile)
	}

	var idx indexFile
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if idx.Tasks == nil {
		return nil, nil
	}

	tasks := make([]model.TaskInput, 0, len(idx.Tasks))
	for _, entry := range idx.Tasks {
		// Accept either "id" or "task_id" field from JSON
		taskID := entry.TaskID
		if taskID == "" {
			taskID = entry.ID
		}
		if taskID == "" {
			return nil, fmt.Errorf("missing task_id: %w", ErrInvalidFile)
		}
		if entry.Title == "" {
			return nil, fmt.Errorf("missing title for task %s: %w", taskID, ErrInvalidFile)
		}

		priority := entry.Priority
		if priority == "" {
			priority = "P1"
		}

		tasks = append(tasks, model.TaskInput{
			TaskID:       taskID,
			Title:        entry.Title,
			Description:  entry.Description,
			Priority:     priority,
			Tags:         entry.Tags,
			Dependencies: entry.Dependencies,
		})
	}

	return tasks, nil
}

// ---------------------------------------------------------------------------
// ParseProposalMD
// ---------------------------------------------------------------------------

func (p *parser) ParseProposalMD(slug string, data []byte) (*model.ProposalInput, error) {
	title := extractFirstH1(data)

	return &model.ProposalInput{
		Slug:    slug,
		Title:   title,
		Content: string(data),
	}, nil
}

// extractFirstH1 scans lines to find the first H1 heading (# Title).
// Returns the trimmed title text without the "# " prefix.
func extractFirstH1(data []byte) string {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		trimmed := bytes.TrimRight(line, " \t")
		if bytes.HasPrefix(trimmed, []byte("# ")) && len(trimmed) > 2 {
			return strings.TrimSpace(string(trimmed[2:]))
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// ParseManifestMD
// ---------------------------------------------------------------------------

func (p *parser) ParseManifestMD(data []byte) (*model.FeatureInput, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data: %w", ErrInvalidFile)
	}

	content := string(data)
	feature, status, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	if feature == "" {
		return nil, fmt.Errorf("missing feature field in frontmatter: %w", ErrInvalidFile)
	}
	if status == "" {
		return nil, fmt.Errorf("missing status field in frontmatter: %w", ErrInvalidFile)
	}

	return &model.FeatureInput{
		Slug:    feature,
		Name:    feature,
		Status:  status,
		Content: content,
	}, nil
}

// parseFrontmatter extracts YAML frontmatter from a Markdown file.
// Frontmatter is enclosed between "---" delimiters at the start of the file.
func parseFrontmatter(content string) (feature, status string, err error) {
	// Frontmatter must start at the very beginning with "---"
	if !strings.HasPrefix(content, "---") {
		return "", "", fmt.Errorf("no frontmatter found: %w", ErrInvalidFile)
	}

	// Find the closing "---"
	afterFirst := content[3:]
	closeIdx := strings.Index(afterFirst, "\n---")
	if closeIdx < 0 {
		return "", "", fmt.Errorf("unclosed frontmatter: %w", ErrInvalidFile)
	}

	fmContent := afterFirst[:closeIdx]

	// Simple YAML parsing for flat key: value pairs
	lines := strings.Split(fmContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := splitYAMLKeyValue(line)
		if !ok {
			continue
		}
		switch key {
		case "feature":
			feature = value
		case "status":
			status = value
		}
	}

	return feature, status, nil
}

// splitYAMLKeyValue splits a YAML "key: value" line.
func splitYAMLKeyValue(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	// Remove surrounding quotes if present
	value = strings.Trim(value, "\"'")
	return key, value, true
}
