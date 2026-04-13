package parser

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	return data
}

// ---------------------------------------------------------------------------
// ParseIndexJSON
// ---------------------------------------------------------------------------

func TestParseIndexJSON_ValidFile(t *testing.T) {
	p := New()
	data := loadTestdata(t, "valid_index.json")

	tasks, err := p.ParseIndexJSON(data)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)

	// Check first task has expected fields
	var found11, found12, found23 bool
	for _, task := range tasks {
		switch task.TaskID {
		case "1.1":
			found11 = true
			assert.Equal(t, "Init monorepo scaffold", task.Title)
			assert.Equal(t, "P0", task.Priority)
			assert.Empty(t, task.Dependencies)
		case "1.2":
			found12 = true
			assert.Equal(t, "Config & DB connection setup", task.Title)
			assert.Equal(t, "P0", task.Priority)
			assert.Equal(t, []string{"1.1"}, task.Dependencies)
		case "2.3":
			found23 = true
			assert.Equal(t, "File parsers", task.Title)
			assert.Equal(t, "P1", task.Priority) // default priority
			assert.Equal(t, []string{"2.2"}, task.Dependencies)
			assert.Equal(t, []string{"parser", "core"}, task.Tags)
		}
	}
	assert.True(t, found11, "task 1.1 should be present")
	assert.True(t, found12, "task 1.2 should be present")
	assert.True(t, found23, "task 2.3 should be present")
}

func TestParseIndexJSON_EmptyTasks(t *testing.T) {
	p := New()
	data := loadTestdata(t, "empty_tasks_index.json")

	tasks, err := p.ParseIndexJSON(data)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestParseIndexJSON_MissingTaskID(t *testing.T) {
	p := New()
	data := loadTestdata(t, "missing_task_id_index.json")

	_, err := p.ParseIndexJSON(data)
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestParseIndexJSON_MissingTitle(t *testing.T) {
	p := New()
	data := loadTestdata(t, "missing_title_index.json")

	_, err := p.ParseIndexJSON(data)
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestParseIndexJSON_InvalidJSON(t *testing.T) {
	p := New()
	data := loadTestdata(t, "invalid_json.txt")

	_, err := p.ParseIndexJSON(data)
	assert.Error(t, err)
}

func TestParseIndexJSON_MissingTasksField(t *testing.T) {
	p := New()
	data := loadTestdata(t, "missing_tasks_field_index.json")

	tasks, err := p.ParseIndexJSON(data)
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestParseIndexJSON_DefaultPriority(t *testing.T) {
	p := New()
	// task with no priority field should default to P1
	data := []byte(`{
		"tasks": {
			"test-task": {
				"task_id": "1.0",
				"title": "Test Task"
			}
		}
	}`)

	tasks, err := p.ParseIndexJSON(data)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "P1", tasks[0].Priority)
}

func TestParseIndexJSON_NilData(t *testing.T) {
	p := New()
	_, err := p.ParseIndexJSON(nil)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// ParseProposalMD
// ---------------------------------------------------------------------------

func TestParseProposalMD_ValidFile(t *testing.T) {
	p := New()
	data := loadTestdata(t, "valid_proposal.md")

	result, err := p.ParseProposalMD("my-feature", data)
	require.NoError(t, err)
	assert.Equal(t, "my-feature", result.Slug)
	assert.Equal(t, "My Awesome Proposal", result.Title)
	assert.Equal(t, string(data), result.Content)
}

func TestParseProposalMD_NoH1(t *testing.T) {
	p := New()
	data := loadTestdata(t, "no_h1_proposal.md")

	result, err := p.ParseProposalMD("test", data)
	require.NoError(t, err)
	assert.Equal(t, "test", result.Slug)
	assert.Empty(t, result.Title, "title should be empty when no H1 found")
	assert.Equal(t, string(data), result.Content)
}

func TestParseProposalMD_EmptyContent(t *testing.T) {
	p := New()

	result, err := p.ParseProposalMD("empty-slug", []byte(""))
	require.NoError(t, err)
	assert.Equal(t, "empty-slug", result.Slug)
	assert.Empty(t, result.Title)
	assert.Empty(t, result.Content)
}

func TestParseProposalMD_H1WithTrailingSpaces(t *testing.T) {
	p := New()
	data := []byte("#   Hello World   \n\nSome content")

	result, err := p.ParseProposalMD("test", data)
	require.NoError(t, err)
	assert.Equal(t, "Hello World", result.Title)
}

func TestParseProposalMD_IgnoresH2(t *testing.T) {
	p := New()
	data := []byte("## Not a title\n\n## Another H2\n\n# Real Title\n\nContent")

	result, err := p.ParseProposalMD("test", data)
	require.NoError(t, err)
	assert.Equal(t, "Real Title", result.Title)
}

// ---------------------------------------------------------------------------
// ParseManifestMD
// ---------------------------------------------------------------------------

func TestParseManifestMD_ValidFile(t *testing.T) {
	p := New()
	data := loadTestdata(t, "valid_manifest.md")

	result, err := p.ParseManifestMD(data)
	require.NoError(t, err)
	assert.Equal(t, "agent-task-center", result.Slug)
	assert.Equal(t, "agent-task-center", result.Name)
	assert.Equal(t, "Draft", result.Status)
	assert.Equal(t, string(data), result.Content)
}

func TestParseManifestMD_NoFrontmatter(t *testing.T) {
	p := New()
	data := loadTestdata(t, "no_frontmatter_manifest.md")

	_, err := p.ParseManifestMD(data)
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestParseManifestMD_EmptyContent(t *testing.T) {
	p := New()
	_, err := p.ParseManifestMD([]byte(""))
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestParseManifestMD_FrontmatterMissingFeature(t *testing.T) {
	p := New()
	data := []byte("---\nstatus: Draft\n---\n\n# Title\n\nContent")

	_, err := p.ParseManifestMD(data)
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestParseManifestMD_FrontmatterMissingStatus(t *testing.T) {
	p := New()
	data := []byte("---\nfeature: my-feature\n---\n\n# Title\n\nContent")

	_, err := p.ParseManifestMD(data)
	assert.ErrorIs(t, err, ErrInvalidFile)
}

func TestParseManifestMD_NilData(t *testing.T) {
	p := New()
	_, err := p.ParseManifestMD(nil)
	assert.ErrorIs(t, err, ErrInvalidFile)
}

// ---------------------------------------------------------------------------
// Interface compliance
// ---------------------------------------------------------------------------

func TestParserImplementsInterface(_ *testing.T) {
	// Compile-time check that concrete type satisfies the interface
	var _ Parser = New() //nolint:staticcheck // QF1011: explicit type assertion for interface compliance check
}
