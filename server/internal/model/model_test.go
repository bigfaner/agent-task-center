package model

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"agent-task-center/server/internal/service"
)

// ---------------------------------------------------------------------------
// DB Entity struct field tag tests
// ---------------------------------------------------------------------------

func TestProjectDBTags(t *testing.T) {
	p := Project{}
	typ := reflect.TypeOf(p)

	expectedDBTags := map[string]string{
		"ID":        "id",
		"Name":      "name",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
	}
	for field, tag := range expectedDBTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("db"), "Project.%s db tag", field)
	}
}

func TestProposalDBTags(t *testing.T) {
	p := Proposal{}
	typ := reflect.TypeOf(p)

	expectedDBTags := map[string]string{
		"ID":        "id",
		"ProjectID": "project_id",
		"Slug":      "slug",
		"Title":     "title",
		"Content":   "content",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
	}
	for field, tag := range expectedDBTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("db"), "Proposal.%s db tag", field)
	}
}

func TestFeatureDBTags(t *testing.T) {
	f := Feature{}
	typ := reflect.TypeOf(f)

	expectedDBTags := map[string]string{
		"ID":        "id",
		"ProjectID": "project_id",
		"Slug":      "slug",
		"Name":      "name",
		"Status":    "status",
		"Content":   "content",
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
	}
	for field, tag := range expectedDBTags {
		ff, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, ff.Tag.Get("db"), "Feature.%s db tag", field)
	}
}

func TestTaskDBTags(t *testing.T) {
	task := Task{}
	typ := reflect.TypeOf(task)

	expectedDBTags := map[string]string{
		"ID":           "id",
		"FeatureID":    "feature_id",
		"TaskID":       "task_id",
		"Title":        "title",
		"Description":  "description",
		"Status":       "status",
		"Priority":     "priority",
		"Tags":         "tags",
		"Dependencies": "dependencies",
		"ClaimedBy":    "claimed_by",
		"Version":      "version",
		"CreatedAt":    "created_at",
		"UpdatedAt":    "updated_at",
	}
	for field, tag := range expectedDBTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("db"), "Task.%s db tag", field)
	}
}

func TestExecutionRecordDBTags(t *testing.T) {
	er := ExecutionRecord{}
	typ := reflect.TypeOf(er)

	expectedDBTags := map[string]string{
		"ID":                 "id",
		"TaskID":             "task_id",
		"AgentID":            "agent_id",
		"Summary":            "summary",
		"FilesCreated":       "files_created",
		"FilesModified":      "files_modified",
		"KeyDecisions":       "key_decisions",
		"TestsPassed":        "tests_passed",
		"TestsFailed":        "tests_failed",
		"Coverage":           "coverage",
		"AcceptanceCriteria": "acceptance_criteria",
		"CreatedAt":          "created_at",
	}
	for field, tag := range expectedDBTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("db"), "ExecutionRecord.%s db tag", field)
	}
}

// ---------------------------------------------------------------------------
// JSON tag tests — verify camelCase json tags on response types
// ---------------------------------------------------------------------------

func TestTaskDetailJSONTags(t *testing.T) {
	td := TaskDetail{}
	typ := reflect.TypeOf(td)

	expectedJSONTags := map[string]string{
		"ID":           "id",
		"TaskID":       "taskId",
		"Title":        "title",
		"Description":  "description",
		"Status":       "status",
		"Priority":     "priority",
		"Tags":         "tags",
		"ClaimedBy":    "claimedBy",
		"Dependencies": "dependencies",
		"CreatedAt":    "createdAt",
		"UpdatedAt":    "updatedAt",
	}
	for field, tag := range expectedJSONTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("json"), "TaskDetail.%s json tag", field)
	}
}

func TestProjectSummaryJSONTags(t *testing.T) {
	ps := ProjectSummary{}
	typ := reflect.TypeOf(ps)

	expectedJSONTags := map[string]string{
		"ID":             "id",
		"Name":           "name",
		"FeatureCount":   "featureCount",
		"TaskTotal":      "taskTotal",
		"CompletionRate": "completionRate",
		"UpdatedAt":      "updatedAt",
	}
	for field, tag := range expectedJSONTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("json"), "ProjectSummary.%s json tag", field)
	}
}

func TestUpsertSummaryJSONTags(t *testing.T) {
	us := UpsertSummary{}
	typ := reflect.TypeOf(us)

	expectedJSONTags := map[string]string{
		"Filename": "filename",
		"Created":  "created",
		"Updated":  "updated",
		"Skipped":  "skipped",
		"Message":  "message",
	}
	for field, tag := range expectedJSONTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("json"), "UpsertSummary.%s json tag", field)
	}
}

// ---------------------------------------------------------------------------
// JSON serialization tests
// ---------------------------------------------------------------------------

func TestTaskDetailJSONSerialization(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	td := TaskDetail{
		ID:           1,
		TaskID:       "2.1",
		Title:        "Create schema",
		Description:  "Set up database tables",
		Status:       "in_progress",
		Priority:     "P0",
		Tags:         []string{"core", "db"},
		ClaimedBy:    "agent-01",
		Dependencies: []string{"1.1", "1.2"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(td)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Verify camelCase keys
	assert.Equal(t, float64(1), result["id"])
	assert.Equal(t, "2.1", result["taskId"])
	assert.Equal(t, "Create schema", result["title"])
	assert.Equal(t, "in_progress", result["status"])
	assert.Equal(t, "P0", result["priority"])
	assert.Equal(t, "agent-01", result["claimedBy"])
	assert.Equal(t, "2026-04-13T12:00:00Z", result["createdAt"])
	assert.Equal(t, "2026-04-13T12:00:00Z", result["updatedAt"])

	// Tags and Dependencies should be arrays
	tags, ok := result["tags"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"core", "db"}, tags)

	deps, ok := result["dependencies"].([]interface{})
	require.True(t, ok)
	assert.Equal(t, []interface{}{"1.1", "1.2"}, deps)
}

func TestProjectSummaryJSONSerialization(t *testing.T) {
	ps := ProjectSummary{
		ID:             1,
		Name:           "my-project",
		FeatureCount:   3,
		TaskTotal:      12,
		CompletionRate: 66.7,
		UpdatedAt:      time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(ps)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, "my-project", result["name"])
	assert.Equal(t, float64(3), result["featureCount"])
	assert.Equal(t, float64(12), result["taskTotal"])
	assert.Equal(t, float64(66.7), result["completionRate"])
	assert.Equal(t, "2026-04-13T00:00:00Z", result["updatedAt"])
}

func TestTaskInputJSONDeserialization(t *testing.T) {
	input := `{
		"task_id": "1.1",
		"title": "Init project",
		"description": "Set up monorepo",
		"priority": "P0",
		"tags": ["core", "setup"],
		"dependencies": []
	}`

	var ti TaskInput
	require.NoError(t, json.Unmarshal([]byte(input), &ti))

	assert.Equal(t, "1.1", ti.TaskID)
	assert.Equal(t, "Init project", ti.Title)
	assert.Equal(t, "Set up monorepo", ti.Description)
	assert.Equal(t, "P0", ti.Priority)
	assert.Equal(t, []string{"core", "setup"}, ti.Tags)
	assert.Empty(t, ti.Dependencies)
}

func TestTaskTagsAndDependenciesNotSerialized(t *testing.T) {
	task := Task{
		ID:           1,
		Tags:         `["core"]`,
		Dependencies: `["1.1"]`,
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))

	// Tags and Dependencies should NOT appear in JSON (json:"-")
	_, hasTags := result["tags"]
	_, hasTagsDash := result["-"]
	_, hasDeps := result["dependencies"]
	_, hasDepsDash := result["-"]
	assert.False(t, hasTags, "Task.Tags should not be serialized to json")
	assert.False(t, hasTagsDash)
	assert.False(t, hasDeps, "Task.Dependencies should not be serialized to json")
	assert.False(t, hasDepsDash)
}

// ---------------------------------------------------------------------------
// Time field RFC3339 serialization
// ---------------------------------------------------------------------------

func TestTimeFieldsSerializeAsRFC3339(t *testing.T) {
	now := time.Date(2026, 4, 13, 15, 30, 45, 0, time.UTC)
	td := TaskDetail{CreatedAt: now, UpdatedAt: now}

	data, err := json.Marshal(td)
	require.NoError(t, err)

	// Verify the time is in RFC3339 format
	assert.Contains(t, string(data), `"createdAt":"2026-04-13T15:30:45Z"`)
	assert.Contains(t, string(data), `"updatedAt":"2026-04-13T15:30:45Z"`)
}

// ---------------------------------------------------------------------------
// TaskFilter tests
// ---------------------------------------------------------------------------

func TestTaskFilterEmpty(t *testing.T) {
	f := TaskFilter{}
	assert.Empty(t, f.Priorities)
	assert.Empty(t, f.Tags)
	assert.Empty(t, f.Statuses)
}

func TestTaskFilterWithValues(t *testing.T) {
	f := TaskFilter{
		Priorities: []string{"P0", "P1"},
		Tags:       []string{"core", "api"},
		Statuses:   []string{"pending", "in_progress"},
	}
	assert.Equal(t, []string{"P0", "P1"}, f.Priorities)
	assert.Equal(t, []string{"core", "api"}, f.Tags)
	assert.Equal(t, []string{"pending", "in_progress"}, f.Statuses)
}

// ---------------------------------------------------------------------------
// Sentinel errors tests (service package)
// ---------------------------------------------------------------------------

func TestSentinelErrors(t *testing.T) {
	errs := map[string]error{
		"ErrNotFound":         service.ErrNotFound,
		"ErrNoAvailableTask":  service.ErrNoAvailableTask,
		"ErrVersionConflict":  service.ErrVersionConflict,
		"ErrInvalidFile":      service.ErrInvalidFile,
		"ErrUnauthorizedAgent": service.ErrUnauthorizedAgent,
	}

	for name, err := range errs {
		t.Run(name, func(t *testing.T) {
			assert.True(t, errors.Is(err, err), "errors.Is should match %s", name)
		})
	}
}

func TestSentinelErrorsDistinct(t *testing.T) {
	// Ensure all sentinel errors are distinct
	all := []error{
		service.ErrNotFound,
		service.ErrNoAvailableTask,
		service.ErrVersionConflict,
		service.ErrInvalidFile,
		service.ErrUnauthorizedAgent,
	}
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			assert.NotEqual(t, all[i], all[j], "errors %d and %d should be distinct", i, j)
		}
	}
}

func TestSentinelErrorsMessages(t *testing.T) {
	assert.Equal(t, "not found", service.ErrNotFound.Error())
	assert.Equal(t, "no available task", service.ErrNoAvailableTask.Error())
	assert.Equal(t, "version conflict", service.ErrVersionConflict.Error())
	assert.Equal(t, "invalid file format", service.ErrInvalidFile.Error())
	assert.Equal(t, "task claimed by different agent", service.ErrUnauthorizedAgent.Error())
}

// ---------------------------------------------------------------------------
// All DB entity fields exist and have correct types
// ---------------------------------------------------------------------------

func TestTaskFieldTypes(t *testing.T) {
	task := Task{}
	typ := reflect.TypeOf(task)

	expectedTypes := map[string]string{
		"ID":           "int64",
		"FeatureID":    "int64",
		"TaskID":       "string",
		"Title":        "string",
		"Description":  "string",
		"Status":       "string",
		"Priority":     "string",
		"Tags":         "string",
		"Dependencies": "string",
		"ClaimedBy":    "string",
		"Version":      "int64",
		"CreatedAt":    "time.Time",
		"UpdatedAt":    "time.Time",
	}
	for field, expected := range expectedTypes {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, expected, f.Type.String(), "Task.%s type", field)
	}
}

func TestExecutionRecordFieldTypes(t *testing.T) {
	er := ExecutionRecord{}
	typ := reflect.TypeOf(er)

	expectedTypes := map[string]string{
		"ID":                 "int64",
		"TaskID":             "int64",
		"AgentID":            "string",
		"Summary":            "string",
		"FilesCreated":       "string",
		"FilesModified":      "string",
		"KeyDecisions":       "string",
		"TestsPassed":        "int",
		"TestsFailed":        "int",
		"Coverage":           "float64",
		"AcceptanceCriteria": "string",
		"CreatedAt":          "time.Time",
	}
	for field, expected := range expectedTypes {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, expected, f.Type.String(), "ExecutionRecord.%s type", field)
	}
}

// ---------------------------------------------------------------------------
// FeatureSummary and ProposalSummary JSON tags
// ---------------------------------------------------------------------------

func TestFeatureSummaryJSONTags(t *testing.T) {
	fs := FeatureSummary{}
	typ := reflect.TypeOf(fs)

	expectedJSONTags := map[string]string{
		"ID":             "id",
		"Slug":           "slug",
		"Name":           "name",
		"Status":         "status",
		"CompletionRate": "completionRate",
		"UpdatedAt":      "updatedAt",
	}
	for field, tag := range expectedJSONTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("json"), "FeatureSummary.%s json tag", field)
	}
}

func TestProposalSummaryJSONTags(t *testing.T) {
	ps := ProposalSummary{}
	typ := reflect.TypeOf(ps)

	expectedJSONTags := map[string]string{
		"ID":           "id",
		"Slug":         "slug",
		"Title":        "title",
		"CreatedAt":    "createdAt",
		"FeatureCount": "featureCount",
	}
	for field, tag := range expectedJSONTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("json"), "ProposalSummary.%s json tag", field)
	}
}

func TestProjectDetailJSONTags(t *testing.T) {
	pd := ProjectDetail{}
	typ := reflect.TypeOf(pd)

	expectedJSONTags := map[string]string{
		"ID":        "id",
		"Name":      "name",
		"Proposals": "proposals",
		"Features":  "features",
	}
	for field, tag := range expectedJSONTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("json"), "ProjectDetail.%s json tag", field)
	}
}

// ---------------------------------------------------------------------------
// TaskInput required fields verification
// ---------------------------------------------------------------------------

func TestTaskInputContainsRequiredFields(t *testing.T) {
	input := TaskInput{
		TaskID:       "2.1",
		Title:        "DB Schema",
		Description:  "Create tables",
		Priority:     "P0",
		Tags:         []string{"db"},
		Dependencies: []string{"1.1"},
	}

	typ := reflect.TypeOf(input)

	requiredFields := []string{"TaskID", "Title", "Description", "Priority", "Tags", "Dependencies"}
	for _, field := range requiredFields {
		_, ok := typ.FieldByName(field)
		assert.True(t, ok, "TaskInput should have field %s", field)
	}

	// Verify json tags for TaskInput
	expectedJSONTags := map[string]string{
		"TaskID":       "task_id",
		"Title":        "title",
		"Description":  "description",
		"Priority":     "priority",
		"Tags":         "tags",
		"Dependencies": "dependencies",
	}
	for field, tag := range expectedJSONTags {
		f, ok := typ.FieldByName(field)
		require.True(t, ok, "field %s not found", field)
		assert.Equal(t, tag, f.Tag.Get("json"), "TaskInput.%s json tag", field)
	}
}

// ---------------------------------------------------------------------------
// Verify Task.Tags and Task.Dependencies are string type (JSON stored)
// ---------------------------------------------------------------------------

func TestTaskTagsAndDependenciesAreStrings(t *testing.T) {
	task := Task{
		Tags:         `["core","api"]`,
		Dependencies: `["1.1","1.2"]`,
	}

	// Verify they are valid JSON
	var tags []string
	assert.NoError(t, json.Unmarshal([]byte(task.Tags), &tags))
	assert.Equal(t, []string{"core", "api"}, tags)

	var deps []string
	assert.NoError(t, json.Unmarshal([]byte(task.Dependencies), &deps))
	assert.Equal(t, []string{"1.1", "1.2"}, deps)
}

// ---------------------------------------------------------------------------
// Verify all json tags use camelCase (no snake_case)
// ---------------------------------------------------------------------------

func TestAllJSONTagsUseCamelCase(t *testing.T) {
	types := []reflect.Type{
		reflect.TypeOf(ProjectSummary{}),
		reflect.TypeOf(ProjectDetail{}),
		reflect.TypeOf(ProposalSummary{}),
		reflect.TypeOf(FeatureSummary{}),
		reflect.TypeOf(TaskDetail{}),
		reflect.TypeOf(UpsertSummary{}),
	}

	for _, typ := range types {
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			tag := f.Tag.Get("json")
			if tag == "" || tag == "-" {
				continue
			}
			name := strings.Split(tag, ",")[0]
			assert.NotContains(t, name, "_", "%s.%s json tag should be camelCase, got %q", typ.Name(), f.Name, name)
		}
	}
}
