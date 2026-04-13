package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"agent-task-center/server/internal/db"
	"agent-task-center/server/internal/model"
	"agent-task-center/server/internal/parser"

	"github.com/jmoiron/sqlx"
)

// UploadFile represents a single file in a push batch.
type UploadFile struct {
	Path     string // full file path, e.g. "docs/features/my-slug/tasks/index.json"
	Filename string // base filename, e.g. "index.json"
	Content  []byte // raw file content
}

// UpsertSummary reports the result of a file parse-and-upsert operation.
// Alias for model.UpsertSummary for convenience within the service package.
type UpsertSummary = model.UpsertSummary

// UploadService handles parsing uploaded files and upserting their contents into the database.
type UploadService interface {
	// ParseAndUpsert parses a single file and upserts its contents.
	// filename determines the routing: index.json, proposal.md, or manifest.md.
	ParseAndUpsert(ctx context.Context, projectName, featureSlug, filename string, content []byte) (*model.UpsertSummary, error)
	// PushDocs processes a batch of files for a project, extracting feature slugs from file paths.
	PushDocs(ctx context.Context, projectName string, files []UploadFile) ([]model.UpsertSummary, error)
}

type uploadService struct {
	db  *sqlx.DB
	prs parser.Parser
}

// NewUploadService creates a new UploadService backed by the given database.
func NewUploadService(d *sqlx.DB) UploadService {
	return &uploadService{db: d, prs: parser.New()}
}

// featureSlugRegex extracts the feature slug from a path like "docs/features/<slug>/tasks/index.json".
var featureSlugRegex = regexp.MustCompile(`docs/features/([^/]+)/`)

// parseAndUpsertManifest handles manifest.md files.
func (s *uploadService) parseAndUpsertManifest(ctx context.Context, projectID int64, content []byte) (*model.UpsertSummary, error) {
	input, err := s.prs.ParseManifestMD(content)
	if err != nil {
		return nil, fmt.Errorf("parse manifest.md: %w", ErrInvalidFile)
	}

	existing, err := db.GetFeatureBySlug(ctx, s.db, projectID, input.Slug)
	if err != nil && !isNotFound(err) {
		return nil, fmt.Errorf("get feature: %w", err)
	}

	_, err = db.UpsertFeature(ctx, s.db, projectID, *input)
	if err != nil {
		return nil, fmt.Errorf("upsert feature: %w", err)
	}

	created := 1
	updated := 0
	if existing != nil {
		created = 0
		updated = 1
	}

	return &model.UpsertSummary{
		Filename: "manifest.md",
		Created:  created,
		Updated:  updated,
		Message:  summarizeManifest(created, updated),
	}, nil
}

// parseAndUpsertProposal handles proposal.md files.
func (s *uploadService) parseAndUpsertProposal(ctx context.Context, projectID int64, slug string, content []byte) (*model.UpsertSummary, error) {
	input, err := s.prs.ParseProposalMD(slug, content)
	if err != nil {
		return nil, fmt.Errorf("parse proposal.md: %w", ErrInvalidFile)
	}

	// Check if proposal already exists
	proposals, err := db.ListProposalsByProject(ctx, s.db, projectID)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}

	var existingProposal *model.Proposal
	for i := range proposals {
		if proposals[i].Slug == slug {
			existingProposal = &proposals[i]
			break
		}
	}

	_, err = db.UpsertProposal(ctx, s.db, projectID, *input)
	if err != nil {
		return nil, fmt.Errorf("upsert proposal: %w", err)
	}

	created := 1
	updated := 0
	if existingProposal != nil {
		created = 0
		updated = 1
	}

	return &model.UpsertSummary{
		Filename: "proposal.md",
		Created:  created,
		Updated:  updated,
		Message:  summarizeProposal(created, updated),
	}, nil
}

// parseAndUpsertIndexJSON handles index.json files.
func (s *uploadService) parseAndUpsertIndexJSON(ctx context.Context, projectID int64, featureSlug string, content []byte) (*model.UpsertSummary, error) {
	tasks, err := s.prs.ParseIndexJSON(content)
	if err != nil {
		return nil, fmt.Errorf("parse index.json: %w", ErrInvalidFile)
	}

	if len(tasks) == 0 {
		return &model.UpsertSummary{
			Filename: "index.json",
			Skipped:  1,
			Message:  "无任务可处理",
		}, nil
	}

	// Ensure the feature exists (auto-create if not)
	feature, err := db.GetFeatureBySlug(ctx, s.db, projectID, featureSlug)
	if err != nil {
		if !isNotFound(err) {
			return nil, fmt.Errorf("get feature: %w", err)
		}
		// Auto-create feature
		feature, err = db.UpsertFeature(ctx, s.db, projectID, model.FeatureInput{
			Slug:    featureSlug,
			Name:    featureSlug,
			Status:  "pending",
			Content: "",
		})
		if err != nil {
			return nil, fmt.Errorf("auto-create feature: %w", err)
		}
	}

	var created, updated int
	for _, task := range tasks {
		existing, err := db.GetTaskByTaskID(ctx, s.db, feature.ID, task.TaskID)
		if err != nil && !isNotFound(err) {
			return nil, fmt.Errorf("get task %s: %w", task.TaskID, err)
		}

		_, err = db.UpsertTask(ctx, s.db, feature.ID, task)
		if err != nil {
			return nil, fmt.Errorf("upsert task %s: %w", task.TaskID, err)
		}

		if existing != nil {
			updated++
		} else {
			created++
		}
	}

	return &model.UpsertSummary{
		Filename: "index.json",
		Created:  created,
		Updated:  updated,
		Message:  fmt.Sprintf("新增 %d 个任务，更新 %d 个任务", created, updated),
	}, nil
}

// ParseAndUpsert routes the file to the correct parser and upsert logic.
func (s *uploadService) ParseAndUpsert(ctx context.Context, projectName, featureSlug, filename string, content []byte) (*model.UpsertSummary, error) {
	proj, err := db.GetOrCreateProject(ctx, s.db, projectName)
	if err != nil {
		return nil, fmt.Errorf("get or create project: %w", err)
	}

	switch strings.ToLower(filename) {
	case "index.json":
		return s.parseAndUpsertIndexJSON(ctx, proj.ID, featureSlug, content)
	case "proposal.md":
		return s.parseAndUpsertProposal(ctx, proj.ID, featureSlug, content)
	case "manifest.md":
		return s.parseAndUpsertManifest(ctx, proj.ID, content)
	default:
		return nil, fmt.Errorf("unsupported file %q: %w", filename, ErrInvalidFile)
	}
}

// PushDocs processes a batch of files, extracting feature slugs from file paths.
func (s *uploadService) PushDocs(ctx context.Context, projectName string, files []UploadFile) ([]model.UpsertSummary, error) {
	summaries := make([]model.UpsertSummary, 0, len(files))

	for _, f := range files {
		featureSlug := extractFeatureSlug(f.Path)

		summary, err := s.ParseAndUpsert(ctx, projectName, featureSlug, f.Filename, f.Content)
		if err != nil {
			// For unsupported files, return a "skipped" summary instead of failing the whole batch
			if isInvalidFile(err) {
				summaries = append(summaries, model.UpsertSummary{
					Filename: f.Filename,
					Skipped:  1,
					Message:  fmt.Sprintf("跳过不支持的文件: %s", f.Filename),
				})
				continue
			}
			return nil, fmt.Errorf("push file %s: %w", f.Path, err)
		}
		summaries = append(summaries, *summary)
	}

	return summaries, nil
}

// extractFeatureSlug extracts the feature slug from a file path.
// Expected format: "docs/features/<slug>/..."
func extractFeatureSlug(path string) string {
	matches := featureSlugRegex.FindStringSubmatch(path)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// isNotFound checks if an error is a "not found" error from the db or service layer.
func isNotFound(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "not found"))
}

// isInvalidFile checks if an error is an ErrInvalidFile.
func isInvalidFile(err error) bool {
	return err != nil && strings.Contains(err.Error(), "invalid file format")
}

// summarizeManifest creates a human-readable summary for manifest.md processing.
func summarizeManifest(created, _ int) string {
	if created > 0 {
		return "新增 Feature"
	}
	return "更新 Feature"
}

// summarizeProposal creates a human-readable summary for proposal.md processing.
func summarizeProposal(created, _ int) string {
	if created > 0 {
		return "新增 Proposal"
	}
	return "更新 Proposal"
}
