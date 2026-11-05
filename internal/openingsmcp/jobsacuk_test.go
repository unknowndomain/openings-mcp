package openingsmcp

import (
	"encoding/json"
	"testing"

	"github.com/amikai/openings-mcp/internal/provider/jobsacuk"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testJobsacukMCPClientServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	srv := jobsacuk.NewMockServer()
	t.Cleanup(srv.Close)
	c := jobsacuk.NewClient(srv.URL, srv.Client())

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	RegisterJobsacuk(server, c)

	t1, t2 := mcp.NewInMemoryTransports()
	ss, err := server.Connect(t.Context(), t1, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ss.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(t.Context(), t2, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func TestRegisterJobsacuk(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	RegisterJobsacuk(server, jobsacuk.NewClient("https://www.jobs.ac.uk", nil))
	assertTools(t, server, "jobsacuk_search_jobs", "jobsacuk_get_job_detail")
}

func TestJobsacukSearchTool(t *testing.T) {
	cs := testJobsacukMCPClientServer(t)
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "jobsacuk_search_jobs",
		Arguments: map[string]any{
			"keyword":   "technical manager",
			"locations": []string{"london"},
			"sort":      "date_placed",
			"page":      2,
		},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	data, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out jobsacukSearchOutput
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, 86, out.Total)
	assert.Equal(t, 2, out.Page)
	require.NotEmpty(t, out.Jobs)
	assert.Equal(t, "Imperial College London", out.Jobs[0].Employer)
}

func TestJobsacukSearchToolRejectsBadSort(t *testing.T) {
	cs := testJobsacukMCPClientServer(t)
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "jobsacuk_search_jobs",
		Arguments: map[string]any{"keyword": "x", "sort": "newest"},
	})
	if err == nil {
		assert.True(t, res.IsError)
	}
}

func TestJobsacukDetailTool(t *testing.T) {
	cs := testJobsacukMCPClientServer(t)
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "jobsacuk_get_job_detail",
		Arguments: map[string]any{"id": "DTJ660"},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)

	data, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out jobsacuk.JobDetail
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, "DTJ660", out.ID)
	assert.Equal(t, "Technical Services Manager", out.Title)
	assert.Equal(t, "2026-11-08", out.ClosesAt)
	assert.NotEmpty(t, out.ApplyURL)
}

func TestJobsacukDetailToolNotFound(t *testing.T) {
	cs := testJobsacukMCPClientServer(t)
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "jobsacuk_get_job_detail",
		Arguments: map[string]any{"id": "ZZZ999"},
	})
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
