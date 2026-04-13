package service

import (
	"context"
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
