// Package db provides database query functions for all entities.
package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-task-center/server/internal/model"

	"github.com/jmoiron/sqlx"
)

// Sentinel errors for the db query layer.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrNoAvailableTask indicates no unclaimed task with met dependencies is available.
	ErrNoAvailableTask = errors.New("no available task")

	// ErrVersionConflict indicates an optimistic locking CAS failure.
	ErrVersionConflict = errors.New("version conflict")

	// ErrUnauthorizedAgent indicates an agent attempted to modify a task claimed by a different agent.
	ErrUnauthorizedAgent = errors.New("task claimed by different agent")
)

func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

// GetOrCreateProject returns an existing project by name, or creates a new one.
func GetOrCreateProject(ctx context.Context, db *sqlx.DB, name string) (*model.Project, error) {
	var p model.Project
	err := db.GetContext(ctx, &p, "SELECT * FROM projects WHERE name = ?", name)
	if err == nil {
		return &p, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get project by name: %w", err)
	}

	now := time.Now()
	result, err := db.ExecContext(ctx,
		"INSERT INTO projects (name, created_at, updated_at) VALUES (?, ?, ?)",
		name, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	return &model.Project{
		ID:        id,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ListProjects returns a paginated list of projects matching the optional search string.
// Returns the projects, total count, and any error.
func ListProjects(ctx context.Context, db *sqlx.DB, search string, page, pageSize int) ([]model.Project, int, error) {
	var total int
	countQuery := "SELECT count(*) FROM projects"
	listQuery := "SELECT * FROM projects"

	var args []interface{}
	if search != "" {
		where := " WHERE name LIKE ?"
		countQuery += where
		listQuery += where
		args = append(args, "%"+search+"%")
	}

	if err := db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	offset := (page - 1) * pageSize
	listQuery += " ORDER BY updated_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	var projects []model.Project
	if err := db.SelectContext(ctx, &projects, listQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}

	return projects, total, nil
}

// GetProject returns a single project by ID.
func GetProject(ctx context.Context, db *sqlx.DB, id int64) (*model.Project, error) {
	var p model.Project
	err := db.GetContext(ctx, &p, "SELECT * FROM projects WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &p, nil
}

// ---------------------------------------------------------------------------
// Proposals
// ---------------------------------------------------------------------------

// UpsertProposal inserts or updates a proposal by (project_id, slug).
func UpsertProposal(ctx context.Context, db *sqlx.DB, projectID int64, input model.ProposalInput) (*model.Proposal, error) {
	now := time.Now()

	// Try to find existing proposal
	var existing model.Proposal
	err := db.GetContext(ctx, &existing,
		"SELECT * FROM proposals WHERE project_id = ? AND slug = ?",
		projectID, input.Slug)

	if errors.Is(err, sql.ErrNoRows) {
		// Insert new
		result, err := db.ExecContext(ctx,
			`INSERT INTO proposals (project_id, slug, title, content, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			projectID, input.Slug, input.Title, input.Content, now, now)
		if err != nil {
			return nil, fmt.Errorf("insert proposal: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("last insert id: %w", err)
		}
		return &model.Proposal{
			ID:        id,
			ProjectID: projectID,
			Slug:      input.Slug,
			Title:     input.Title,
			Content:   input.Content,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}

	// Update existing
	_, err = db.ExecContext(ctx,
		`UPDATE proposals SET title = ?, content = ?, updated_at = ?
		 WHERE id = ?`,
		input.Title, input.Content, now, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("update proposal: %w", err)
	}

	existing.Title = input.Title
	existing.Content = input.Content
	existing.UpdatedAt = now
	return &existing, nil
}

// ListProposalsByProject returns all proposals for a given project.
func ListProposalsByProject(ctx context.Context, db *sqlx.DB, projectID int64) ([]model.Proposal, error) {
	var proposals []model.Proposal
	err := db.SelectContext(ctx, &proposals,
		"SELECT * FROM proposals WHERE project_id = ? ORDER BY created_at ASC",
		projectID)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	return proposals, nil
}

// GetProposal returns a single proposal by ID.
func GetProposal(ctx context.Context, db *sqlx.DB, id int64) (*model.Proposal, error) {
	var p model.Proposal
	err := db.GetContext(ctx, &p, "SELECT * FROM proposals WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get proposal: %w", err)
	}
	return &p, nil
}

// ---------------------------------------------------------------------------
// Features
// ---------------------------------------------------------------------------

// UpsertFeature inserts or updates a feature by (project_id, slug).
func UpsertFeature(ctx context.Context, db *sqlx.DB, projectID int64, input model.FeatureInput) (*model.Feature, error) {
	now := time.Now()

	// Try to find existing feature
	var existing model.Feature
	err := db.GetContext(ctx, &existing,
		"SELECT * FROM features WHERE project_id = ? AND slug = ?",
		projectID, input.Slug)

	if errors.Is(err, sql.ErrNoRows) {
		// Insert new
		result, err := db.ExecContext(ctx,
			`INSERT INTO features (project_id, slug, name, status, content, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			projectID, input.Slug, input.Name, input.Status, input.Content, now, now)
		if err != nil {
			return nil, fmt.Errorf("insert feature: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("last insert id: %w", err)
		}
		return &model.Feature{
			ID:        id,
			ProjectID: projectID,
			Slug:      input.Slug,
			Name:      input.Name,
			Status:    input.Status,
			Content:   input.Content,
			CreatedAt: now,
			UpdatedAt: now,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get feature: %w", err)
	}

	// Update existing
	_, err = db.ExecContext(ctx,
		`UPDATE features SET name = ?, status = ?, content = ?, updated_at = ?
		 WHERE id = ?`,
		input.Name, input.Status, input.Content, now, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("update feature: %w", err)
	}

	existing.Name = input.Name
	existing.Status = input.Status
	existing.Content = input.Content
	existing.UpdatedAt = now
	return &existing, nil
}

// ListFeaturesByProject returns all features for a given project.
func ListFeaturesByProject(ctx context.Context, db *sqlx.DB, projectID int64) ([]model.Feature, error) {
	var features []model.Feature
	err := db.SelectContext(ctx, &features,
		"SELECT * FROM features WHERE project_id = ? ORDER BY created_at ASC",
		projectID)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}
	return features, nil
}

// GetFeature returns a single feature by ID.
func GetFeature(ctx context.Context, db *sqlx.DB, id int64) (*model.Feature, error) {
	var f model.Feature
	err := db.GetContext(ctx, &f, "SELECT * FROM features WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get feature: %w", err)
	}
	return &f, nil
}

// GetFeatureBySlug returns a single feature by project ID and slug.
func GetFeatureBySlug(ctx context.Context, db *sqlx.DB, projectID int64, slug string) (*model.Feature, error) {
	var f model.Feature
	err := db.GetContext(ctx, &f,
		"SELECT * FROM features WHERE project_id = ? AND slug = ?",
		projectID, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get feature by slug: %w", err)
	}
	return &f, nil
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

// UpsertTask inserts or updates a task by (feature_id, task_id).
// It does NOT overwrite status, claimed_by, or version fields.
func UpsertTask(ctx context.Context, db *sqlx.DB, featureID int64, input model.TaskInput) (*model.Task, error) {
	tagsJSON := "[]"
	if len(input.Tags) > 0 {
		b, _ := jsonMarshal(input.Tags)
		tagsJSON = string(b)
	}

	depsJSON := "[]"
	if len(input.Dependencies) > 0 {
		b, _ := jsonMarshal(input.Dependencies)
		depsJSON = string(b)
	}

	priority := input.Priority
	if priority == "" {
		priority = "P1"
	}

	now := time.Now()

	// Try to find existing task
	var existing model.Task
	err := db.GetContext(ctx, &existing,
		"SELECT * FROM tasks WHERE feature_id = ? AND task_id = ?",
		featureID, input.TaskID)

	if errors.Is(err, sql.ErrNoRows) {
		// Insert new task
		result, err := db.ExecContext(ctx,
			`INSERT INTO tasks (feature_id, task_id, title, description, priority, tags, dependencies, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			featureID, input.TaskID, input.Title, input.Description, priority, tagsJSON, depsJSON, now, now)
		if err != nil {
			return nil, fmt.Errorf("insert task: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("last insert id: %w", err)
		}
		return &model.Task{
			ID:           id,
			FeatureID:    featureID,
			TaskID:       input.TaskID,
			Title:        input.Title,
			Description:  input.Description,
			Status:       "pending",
			Priority:     priority,
			Tags:         tagsJSON,
			Dependencies: depsJSON,
			ClaimedBy:    "",
			Version:      0,
			CreatedAt:    now,
			UpdatedAt:    now,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	// Update existing task — preserve status, claimed_by, version
	_, err = db.ExecContext(ctx,
		`UPDATE tasks SET title = ?, description = ?, priority = ?, tags = ?, dependencies = ?, updated_at = ?
		 WHERE id = ?`,
		input.Title, input.Description, priority, tagsJSON, depsJSON, now, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	existing.Title = input.Title
	existing.Description = input.Description
	existing.Priority = priority
	existing.Tags = tagsJSON
	existing.Dependencies = depsJSON
	existing.UpdatedAt = now
	return &existing, nil
}

// ListTasksByFeature returns tasks for a feature with optional filtering.
func ListTasksByFeature(ctx context.Context, db *sqlx.DB, featureID int64, filter model.TaskFilter) ([]model.Task, error) {
	query := "SELECT * FROM tasks WHERE feature_id = ?"
	var args []interface{}
	args = append(args, featureID)

	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			placeholders[i] = "?"
			args = append(args, s)
		}
		query += " AND status IN (" + strings.Join(placeholders, ",") + ")"
	}

	if len(filter.Priorities) > 0 {
		placeholders := make([]string, len(filter.Priorities))
		for i, p := range filter.Priorities {
			placeholders[i] = "?"
			args = append(args, p)
		}
		query += " AND priority IN (" + strings.Join(placeholders, ",") + ")"
	}

	if len(filter.Tags) > 0 {
		for _, tag := range filter.Tags {
			query += " AND tags LIKE ?"
			args = append(args, `%"`+tag+`"%`)
		}
	}

	query += " ORDER BY task_id ASC"

	var tasks []model.Task
	if err := db.SelectContext(ctx, &tasks, query, args...); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

// GetTask returns a single task by its database ID.
func GetTask(ctx context.Context, db *sqlx.DB, id int64) (*model.Task, error) {
	var t model.Task
	err := db.GetContext(ctx, &t, "SELECT * FROM tasks WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return &t, nil
}

// GetTaskByTaskID returns a single task by feature_id and task_id string.
func GetTaskByTaskID(ctx context.Context, db *sqlx.DB, featureID int64, taskID string) (*model.Task, error) {
	var t model.Task
	err := db.GetContext(ctx, &t,
		"SELECT * FROM tasks WHERE feature_id = ? AND task_id = ?",
		featureID, taskID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get task by task_id: %w", err)
	}
	return &t, nil
}

// ClaimTask performs an optimistic-locking claim of a pending task with met dependencies.
// It retries up to 3 times on version conflicts before returning ErrNoAvailableTask.
func ClaimTask(ctx context.Context, db *sqlx.DB, featureID int64, agentID string) (*model.Task, error) {
	// Find candidates: pending tasks with all dependencies completed, ordered by priority
	candidateQuery := `
		SELECT * FROM tasks
		WHERE feature_id = ?
		  AND status = 'pending'
		  AND (
		    dependencies = '[]'
		    OR dependencies = ''
		    OR NOT EXISTS (
		      SELECT 1 FROM json_each(tasks.dependencies) AS dep
		      WHERE dep.value NOT IN (
		        SELECT t2.task_id FROM tasks t2
		        WHERE t2.feature_id = tasks.feature_id AND t2.status = 'completed'
		      )
		    )
		  )
		ORDER BY CASE priority
			WHEN 'P0' THEN 0
			WHEN 'P1' THEN 1
			WHEN 'P2' THEN 2
			ELSE 3
		END, task_id ASC`

	var candidates []model.Task
	if err := db.SelectContext(ctx, &candidates, candidateQuery, featureID); err != nil {
		return nil, fmt.Errorf("find claim candidates: %w", err)
	}

	if len(candidates) == 0 {
		return nil, ErrNoAvailableTask
	}

	maxRetries := 3
	for i := 0; i < maxRetries && i < len(candidates); i++ {
		candidate := candidates[i]
		now := time.Now()

		result, err := db.ExecContext(ctx,
			`UPDATE tasks
			 SET status = 'in_progress', claimed_by = ?, version = version + 1, updated_at = ?
			 WHERE id = ? AND version = ? AND status = 'pending'`,
			agentID, now, candidate.ID, candidate.Version)
		if err != nil {
			return nil, fmt.Errorf("claim task: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("rows affected: %w", err)
		}

		if rowsAffected > 0 {
			candidate.Status = "in_progress"
			candidate.ClaimedBy = agentID
			candidate.Version++
			candidate.UpdatedAt = now
			return &candidate, nil
		}
		// Version conflict, try next candidate
	}

	return nil, ErrNoAvailableTask
}

// UpdateTaskStatus updates a task's status, verifying the agentID matches claimed_by.
func UpdateTaskStatus(ctx context.Context, db *sqlx.DB, id int64, agentID, status string) error {
	var task model.Task
	err := db.GetContext(ctx, &task, "SELECT * FROM tasks WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("get task for status update: %w", err)
	}

	if task.ClaimedBy != agentID {
		return ErrUnauthorizedAgent
	}

	now := time.Now()
	result, err := db.ExecContext(ctx,
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND version = ?`,
		status, now, id, task.Version)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrVersionConflict
	}

	return nil
}

// ---------------------------------------------------------------------------
// Execution Records
// ---------------------------------------------------------------------------

// InsertRecord inserts a new execution record for a task.
func InsertRecord(ctx context.Context, db *sqlx.DB, taskID int64, record model.ExecutionRecord) (*model.ExecutionRecord, error) {
	now := time.Now()

	result, err := db.ExecContext(ctx,
		`INSERT INTO execution_records (
			task_id, agent_id, summary,
			files_created, files_modified, key_decisions,
			tests_passed, tests_failed, coverage,
			acceptance_criteria, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		taskID, record.AgentID, record.Summary,
		record.FilesCreated, record.FilesModified, record.KeyDecisions,
		record.TestsPassed, record.TestsFailed, record.Coverage,
		record.AcceptanceCriteria, now)
	if err != nil {
		return nil, fmt.Errorf("insert execution record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("last insert id: %w", err)
	}

	return &model.ExecutionRecord{
		ID:                 id,
		TaskID:             taskID,
		AgentID:            record.AgentID,
		Summary:            record.Summary,
		FilesCreated:       record.FilesCreated,
		FilesModified:      record.FilesModified,
		KeyDecisions:       record.KeyDecisions,
		TestsPassed:        record.TestsPassed,
		TestsFailed:        record.TestsFailed,
		Coverage:           record.Coverage,
		AcceptanceCriteria: record.AcceptanceCriteria,
		CreatedAt:          now,
	}, nil
}

// ListRecordsByTask returns paginated execution records for a task.
func ListRecordsByTask(ctx context.Context, db *sqlx.DB, taskID int64, page, pageSize int) ([]model.ExecutionRecord, int, error) {
	var total int
	if err := db.GetContext(ctx, &total,
		"SELECT count(*) FROM execution_records WHERE task_id = ?", taskID); err != nil {
		return nil, 0, fmt.Errorf("count records: %w", err)
	}

	offset := (page - 1) * pageSize
	var records []model.ExecutionRecord
	if err := db.SelectContext(ctx, &records,
		"SELECT * FROM execution_records WHERE task_id = ? ORDER BY id DESC LIMIT ? OFFSET ?",
		taskID, pageSize, offset); err != nil {
		return nil, 0, fmt.Errorf("list records: %w", err)
	}

	return records, total, nil
}
