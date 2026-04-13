package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"agent-task-center/server/internal/db"
	"agent-task-center/server/internal/model"

	"github.com/jmoiron/sqlx"
)

// ---------------------------------------------------------------------------
// ProjectService
// ---------------------------------------------------------------------------

// ProjectService handles business logic for projects.
type ProjectService interface {
	List(ctx context.Context, search string, page, pageSize int) ([]model.ProjectSummary, int, error)
	Get(ctx context.Context, id int64) (*model.ProjectDetail, error)
	Upsert(ctx context.Context, name string) (*model.Project, error)
}

type projectService struct {
	db *sqlx.DB
}

// NewProjectService creates a new ProjectService backed by the given database.
func NewProjectService(db *sqlx.DB) ProjectService {
	return &projectService{db: db}
}

func (s *projectService) Upsert(ctx context.Context, name string) (*model.Project, error) {
	return db.GetOrCreateProject(ctx, s.db, name)
}

func (s *projectService) List(ctx context.Context, search string, page, pageSize int) ([]model.ProjectSummary, int, error) {
	projects, total, err := db.ListProjects(ctx, s.db, search, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}

	summaries := make([]model.ProjectSummary, 0, len(projects))
	for _, p := range projects {
		summary := model.ProjectSummary{
			ID:        p.ID,
			Name:      p.Name,
			UpdatedAt: p.UpdatedAt,
		}

		// Count features for this project
		features, err := db.ListFeaturesByProject(ctx, s.db, p.ID)
		if err != nil {
			return nil, 0, fmt.Errorf("list features for project %d: %w", p.ID, err)
		}
		summary.FeatureCount = len(features)

		// Count total and completed tasks across all features
		var taskTotal, completedTotal int
		for _, f := range features {
			tasks, err := db.ListTasksByFeature(ctx, s.db, f.ID, model.TaskFilter{})
			if err != nil {
				return nil, 0, fmt.Errorf("list tasks for feature %d: %w", f.ID, err)
			}
			taskTotal += len(tasks)
			for _, t := range tasks {
				if t.Status == "completed" {
					completedTotal++
				}
			}
		}
		summary.TaskTotal = taskTotal
		if taskTotal > 0 {
			summary.CompletionRate = float64(completedTotal) / float64(taskTotal) * 100
		}

		summaries = append(summaries, summary)
	}

	return summaries, total, nil
}

func (s *projectService) Get(ctx context.Context, id int64) (*model.ProjectDetail, error) {
	p, err := db.GetProject(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}

	// Build proposals
	proposals, err := db.ListProposalsByProject(ctx, s.db, id)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}

	features, err := db.ListFeaturesByProject(ctx, s.db, id)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}

	proposalSummaries := make([]model.ProposalSummary, 0, len(proposals))
	for _, prop := range proposals {
		proposalSummaries = append(proposalSummaries, model.ProposalSummary{
			ID:           prop.ID,
			Slug:         prop.Slug,
			Title:        prop.Title,
			CreatedAt:    prop.CreatedAt,
			FeatureCount: len(features), // features belong to the project
		})
	}

	featureSummaries := make([]model.FeatureSummary, 0, len(features))
	for _, f := range features {
		fs := model.FeatureSummary{
			ID:        f.ID,
			Slug:      f.Slug,
			Name:      f.Name,
			Status:    f.Status,
			UpdatedAt: f.UpdatedAt,
		}

		tasks, err := db.ListTasksByFeature(ctx, s.db, f.ID, model.TaskFilter{})
		if err != nil {
			return nil, fmt.Errorf("list tasks for feature %d: %w", f.ID, err)
		}

		var completed int
		for _, t := range tasks {
			if t.Status == "completed" {
				completed++
			}
		}
		if len(tasks) > 0 {
			fs.CompletionRate = float64(completed) / float64(len(tasks)) * 100
		}

		featureSummaries = append(featureSummaries, fs)
	}

	return &model.ProjectDetail{
		ID:        p.ID,
		Name:      p.Name,
		Proposals: proposalSummaries,
		Features:  featureSummaries,
	}, nil
}

// ---------------------------------------------------------------------------
// FeatureService
// ---------------------------------------------------------------------------

// FeatureService handles business logic for features.
type FeatureService interface {
	ListByProject(ctx context.Context, projectID int64) ([]model.FeatureSummary, error)
	GetTasks(ctx context.Context, featureID int64, filter model.TaskFilter) ([]model.Task, error)
	GetByID(ctx context.Context, id int64) (*model.Feature, error)
}

type featureService struct {
	db *sqlx.DB
}

// NewFeatureService creates a new FeatureService backed by the given database.
func NewFeatureService(db *sqlx.DB) FeatureService {
	return &featureService{db: db}
}

func (s *featureService) ListByProject(ctx context.Context, projectID int64) ([]model.FeatureSummary, error) {
	features, err := db.ListFeaturesByProject(ctx, s.db, projectID)
	if err != nil {
		return nil, fmt.Errorf("list features: %w", err)
	}

	summaries := make([]model.FeatureSummary, 0, len(features))
	for _, f := range features {
		fs := model.FeatureSummary{
			ID:        f.ID,
			Slug:      f.Slug,
			Name:      f.Name,
			Status:    f.Status,
			UpdatedAt: f.UpdatedAt,
		}

		tasks, err := db.ListTasksByFeature(ctx, s.db, f.ID, model.TaskFilter{})
		if err != nil {
			return nil, fmt.Errorf("list tasks for feature %d: %w", f.ID, err)
		}

		var completed int
		for _, t := range tasks {
			if t.Status == "completed" {
				completed++
			}
		}
		if len(tasks) > 0 {
			fs.CompletionRate = float64(completed) / float64(len(tasks)) * 100
		}

		summaries = append(summaries, fs)
	}

	return summaries, nil
}

func (s *featureService) GetTasks(ctx context.Context, featureID int64, filter model.TaskFilter) ([]model.Task, error) {
	return db.ListTasksByFeature(ctx, s.db, featureID, filter)
}

// GetByID returns a feature by its database ID.
func (s *featureService) GetByID(ctx context.Context, id int64) (*model.Feature, error) {
	f, err := db.GetFeature(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get feature: %w", err)
	}
	return f, nil
}

// ---------------------------------------------------------------------------
// ProposalService
// ---------------------------------------------------------------------------

// ProposalService handles business logic for proposals.
type ProposalService interface {
	GetByID(ctx context.Context, id int64) (*model.Proposal, error)
}

type proposalService struct {
	db *sqlx.DB
}

// NewProposalService creates a new ProposalService backed by the given database.
func NewProposalService(db *sqlx.DB) ProposalService {
	return &proposalService{db: db}
}

// GetByID returns a proposal by its database ID.
func (s *proposalService) GetByID(ctx context.Context, id int64) (*model.Proposal, error) {
	p, err := db.GetProposal(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get proposal: %w", err)
	}
	return p, nil
}

// ---------------------------------------------------------------------------
// TaskService
// ---------------------------------------------------------------------------

// validStatuses contains the set of status values accepted by UpdateStatus.
var validStatuses = map[string]bool{
	"in_progress": true,
	"blocked":     true,
	"pending":     true,
}

// TaskService handles business logic for tasks: claim, status update, and record submission.
type TaskService interface {
	Get(ctx context.Context, id int64) (*model.TaskDetail, error)
	GetByTaskID(ctx context.Context, projectName, featureSlug, taskID string) (*model.TaskDetail, error)
	ListRecords(ctx context.Context, taskID int64, page, pageSize int) ([]model.ExecutionRecord, int, error)
	Claim(ctx context.Context, projectName, featureSlug, agentID string) (*model.Task, error)
	UpdateStatus(ctx context.Context, taskID int64, agentID, status string) error
	SubmitRecord(ctx context.Context, taskID int64, agentID string, record model.ExecutionRecord) (*model.ExecutionRecord, error)
}

type taskService struct {
	db *sqlx.DB
}

// NewTaskService creates a new TaskService backed by the given database.
func NewTaskService(db *sqlx.DB) TaskService {
	return &taskService{db: db}
}

// Get returns the full detail of a task by its database ID.
// Tags and Dependencies are deserialized from JSON strings to string slices.
func (s *taskService) Get(ctx context.Context, id int64) (*model.TaskDetail, error) {
	t, err := db.GetTask(ctx, s.db, id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}
	return taskToDetail(t)
}

// GetByTaskID locates a task by the project+feature+taskId triple and returns its full detail.
func (s *taskService) GetByTaskID(ctx context.Context, projectName, featureSlug, taskID string) (*model.TaskDetail, error) {
	proj, err := db.GetOrCreateProject(ctx, s.db, projectName)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	feat, err := db.GetFeatureBySlug(ctx, s.db, proj.ID, featureSlug)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get feature: %w", err)
	}

	t, err := db.GetTaskByTaskID(ctx, s.db, feat.ID, taskID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get task by task_id: %w", err)
	}

	return taskToDetail(t)
}

// ListRecords returns paginated execution records for a task.
func (s *taskService) ListRecords(ctx context.Context, taskID int64, page, pageSize int) ([]model.ExecutionRecord, int, error) {
	return db.ListRecordsByTask(ctx, s.db, taskID, page, pageSize)
}

// Claim finds and claims the highest-priority pending task with met dependencies
// under the specified project+feature, using optimistic locking.
func (s *taskService) Claim(ctx context.Context, projectName, featureSlug, agentID string) (*model.Task, error) {
	proj, err := db.GetOrCreateProject(ctx, s.db, projectName)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	feat, err := db.GetFeatureBySlug(ctx, s.db, proj.ID, featureSlug)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNoAvailableTask
		}
		return nil, fmt.Errorf("get feature: %w", err)
	}

	task, err := db.ClaimTask(ctx, s.db, feat.ID, agentID)
	if err != nil {
		if errors.Is(err, db.ErrNoAvailableTask) {
			return nil, ErrNoAvailableTask
		}
		return nil, fmt.Errorf("claim task: %w", err)
	}

	return task, nil
}

// UpdateStatus changes a task's status after verifying the agent owns it.
// Valid status values are: in_progress, blocked, pending.
func (s *taskService) UpdateStatus(ctx context.Context, taskID int64, agentID, status string) error {
	if !validStatuses[status] {
		return ErrInvalidStatus
	}

	err := db.UpdateTaskStatus(ctx, s.db, taskID, agentID, status)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return ErrNotFound
		}
		if errors.Is(err, db.ErrUnauthorizedAgent) {
			return ErrUnauthorizedAgent
		}
		if errors.Is(err, db.ErrVersionConflict) {
			return ErrVersionConflict
		}
		return fmt.Errorf("update task status: %w", err)
	}

	return nil
}

// SubmitRecord inserts an execution record and marks the task as completed.
// It verifies that the agentID matches the task's claimed_by field.
func (s *taskService) SubmitRecord(ctx context.Context, taskID int64, agentID string, record model.ExecutionRecord) (*model.ExecutionRecord, error) {
	// Verify the task exists and the agent owns it
	t, err := db.GetTask(ctx, s.db, taskID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get task: %w", err)
	}

	if t.ClaimedBy != agentID {
		return nil, ErrUnauthorizedAgent
	}

	// Insert the execution record
	record.AgentID = agentID
	saved, err := db.InsertRecord(ctx, s.db, taskID, record)
	if err != nil {
		return nil, fmt.Errorf("insert record: %w", err)
	}

	// Update task status to completed
	err = db.UpdateTaskStatus(ctx, s.db, taskID, agentID, "completed")
	if err != nil {
		if errors.Is(err, db.ErrUnauthorizedAgent) {
			return nil, ErrUnauthorizedAgent
		}
		if errors.Is(err, db.ErrVersionConflict) {
			return nil, ErrVersionConflict
		}
		return nil, fmt.Errorf("update task to completed: %w", err)
	}

	return saved, nil
}

// taskToDetail converts a model.Task to a model.TaskDetail by deserializing
// the Tags and Dependencies JSON fields.
func taskToDetail(t *model.Task) (*model.TaskDetail, error) {
	var tags []string
	if err := json.Unmarshal([]byte(t.Tags), &tags); err != nil {
		tags = []string{}
	}

	var deps []string
	if err := json.Unmarshal([]byte(t.Dependencies), &deps); err != nil {
		deps = []string{}
	}

	return &model.TaskDetail{
		ID:           t.ID,
		TaskID:       t.TaskID,
		Title:        t.Title,
		Description:  t.Description,
		Status:       t.Status,
		Priority:     t.Priority,
		Tags:         tags,
		ClaimedBy:    t.ClaimedBy,
		Dependencies: deps,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}, nil
}
