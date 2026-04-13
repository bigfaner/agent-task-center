// Package model defines all Go data structures for the Agent Task Center.
// Types are organized into three categories:
//   - DB entities: structs with db tags matching the database schema columns
//   - Parser input types: structs for file parsing layer input
//   - Service aggregation types: structs for API responses with json tags
package model

import (
	"time"
)

// ---------------------------------------------------------------------------
// DB Entity types — map 1:1 to database table rows via db struct tags.
// ---------------------------------------------------------------------------

// Project is the top-level entity, created on first push.
type Project struct {
	ID        int64     `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Proposal represents a proposal.md file within a project.
type Proposal struct {
	ID        int64     `db:"id"         json:"id"`
	ProjectID int64     `db:"project_id" json:"projectId"`
	Slug      string    `db:"slug"       json:"slug"`
	Title     string    `db:"title"      json:"title"`
	Content   string    `db:"content"    json:"content"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Feature represents a manifest.md file within a project.
type Feature struct {
	ID        int64     `db:"id"         json:"id"`
	ProjectID int64     `db:"project_id" json:"projectId"`
	Slug      string    `db:"slug"       json:"slug"`
	Name      string    `db:"name"       json:"name"`
	Status    string    `db:"status"     json:"status"`
	Content   string    `db:"content"    json:"content"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// Task represents a single task entry parsed from index.json.
// Tags and Dependencies are stored as JSON strings in the database.
type Task struct {
	ID           int64     `db:"id"            json:"id"`
	FeatureID    int64     `db:"feature_id"    json:"featureId"`
	TaskID       string    `db:"task_id"       json:"taskId"`
	Title        string    `db:"title"         json:"title"`
	Description  string    `db:"description"   json:"description"`
	Status       string    `db:"status"        json:"status"`
	Priority     string    `db:"priority"      json:"priority"`
	Tags         string    `db:"tags"          json:"-"` // JSON string in DB
	Dependencies string    `db:"dependencies"  json:"-"` // JSON string in DB
	ClaimedBy    string    `db:"claimed_by"    json:"claimedBy"`
	Version      int64     `db:"version"       json:"version"`
	CreatedAt    time.Time `db:"created_at"    json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at"    json:"updatedAt"`
}

// ExecutionRecord is submitted via the task record CLI command.
type ExecutionRecord struct {
	ID                 int64     `db:"id"                  json:"id"`
	TaskID             int64     `db:"task_id"             json:"taskId"`
	AgentID            string    `db:"agent_id"            json:"agentId"`
	Summary            string    `db:"summary"             json:"summary"`
	FilesCreated       string    `db:"files_created"       json:"filesCreated"`
	FilesModified      string    `db:"files_modified"      json:"filesModified"`
	KeyDecisions       string    `db:"key_decisions"       json:"keyDecisions"`
	TestsPassed        int       `db:"tests_passed"        json:"testsPassed"`
	TestsFailed        int       `db:"tests_failed"        json:"testsFailed"`
	Coverage           float64   `db:"coverage"            json:"coverage"`
	AcceptanceCriteria string    `db:"acceptance_criteria" json:"acceptanceCriteria"`
	CreatedAt          time.Time `db:"created_at"          json:"createdAt"`
}

// ---------------------------------------------------------------------------
// Parser input types — used by the parser layer for file parsing results.
// ---------------------------------------------------------------------------

// TaskInput is the parsed representation of a single task from index.json.
type TaskInput struct {
	TaskID       string   `json:"task_id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Priority     string   `json:"priority"`
	Tags         []string `json:"tags"`
	Dependencies []string `json:"dependencies"`
}

// ProposalInput is the parsed representation of a proposal.md file.
type ProposalInput struct {
	Slug    string // derived from directory name in file path
	Title   string // parsed from the first H1 heading in Markdown
	Content string // raw Markdown content
}

// FeatureInput is the parsed representation of a manifest.md file.
type FeatureInput struct {
	Slug    string // from frontmatter feature field
	Name    string // display name (same as slug or frontmatter name field)
	Status  string // from frontmatter status field
	Content string // raw manifest.md content
}

// ---------------------------------------------------------------------------
// Service aggregation types — used for API responses with camelCase json tags.
// ---------------------------------------------------------------------------

// ProjectSummary is the list-view representation of a project.
type ProjectSummary struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	FeatureCount   int       `json:"featureCount"`
	TaskTotal      int       `json:"taskTotal"`
	CompletionRate float64   `json:"completionRate"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ProjectDetail is the full detail view of a project with nested proposals and features.
type ProjectDetail struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Proposals []ProposalSummary `json:"proposals"`
	Features  []FeatureSummary  `json:"features"`
}

// ProposalSummary is a lightweight summary of a proposal.
type ProposalSummary struct {
	ID           int64     `json:"id"`
	Slug         string    `json:"slug"`
	Title        string    `json:"title"`
	CreatedAt    time.Time `json:"createdAt"`
	FeatureCount int       `json:"featureCount"`
}

// FeatureSummary is a lightweight summary of a feature.
type FeatureSummary struct {
	ID             int64     `json:"id"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	CompletionRate float64   `json:"completionRate"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// TaskDetail is the full detail view of a task with deserialized Tags and Dependencies.
type TaskDetail struct {
	ID           int64     `json:"id"`
	TaskID       string    `json:"taskId"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	Tags         []string  `json:"tags"`
	ClaimedBy    string    `json:"claimedBy"`
	Dependencies []string  `json:"dependencies"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// UpsertSummary reports the result of a file parse-and-upsert operation.
type UpsertSummary struct {
	Filename string `json:"filename"`
	Created  int    `json:"created"`
	Updated  int    `json:"updated"`
	Skipped  int    `json:"skipped"`
	Message  string `json:"message"`
}

// ---------------------------------------------------------------------------
// Filter types — used for server-side task filtering.
// ---------------------------------------------------------------------------

// TaskFilter holds optional filter criteria for listing tasks.
// Empty slices mean no filtering on that field.
type TaskFilter struct {
	Priorities []string // e.g. ["P0","P1"]
	Tags       []string // e.g. ["core","api"]
	Statuses   []string // e.g. ["pending","in_progress"]
}
