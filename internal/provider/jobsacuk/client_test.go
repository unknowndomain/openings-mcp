package jobsacuk

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) *Client {
	t.Helper()
	srv := NewMockServer()
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, srv.Client())
}

func TestSearch(t *testing.T) {
	c := newTestClient(t)
	got, err := c.Search(t.Context(), &SearchRequest{Keyword: "technical manager"})
	require.NoError(t, err)

	assert.Equal(t, 262, got.Total)
	assert.Equal(t, 1, got.Page)
	assert.Equal(t, 11, got.TotalPages)
	require.Len(t, got.Jobs, 25)

	first := got.Jobs[0]
	assert.Equal(t, "DTJ660", first.ID)
	assert.Equal(t, "Technical Services Manager", first.Title)
	assert.True(t, strings.HasSuffix(first.URL, "/job/DTJ660/technical-services-manager"), first.URL)
	assert.Equal(t, "University for the Creative Arts", first.Employer)
	assert.Equal(t, "Farnham School of Creative and Cultural Industries: Technical", first.Department)
	assert.Equal(t, "Farnham", first.Location)
	assert.Equal(t, "£48,822 per annum", first.Salary)
	assert.Equal(t, "23 Oct", first.DatePlaced)
	assert.Equal(t, "08 Nov", first.Closes)

	for _, j := range got.Jobs {
		assert.NotEmpty(t, j.ID)
		assert.NotEmpty(t, j.Title)
		assert.NotEmpty(t, j.Employer)
	}
}

func TestSearchFiltered(t *testing.T) {
	c := newTestClient(t)
	got, err := c.Search(t.Context(), &SearchRequest{
		Keyword:   "technical manager",
		Locations: []string{"London"},
		Sort:      SortDatePlaced,
		Page:      2,
	})
	require.NoError(t, err)
	assert.Equal(t, 86, got.Total)
	assert.Equal(t, 2, got.Page)
	assert.Equal(t, 4, got.TotalPages)
	require.NotEmpty(t, got.Jobs)
	assert.Equal(t, "Technical Operations Manager (Electrical & Mechanical)", got.Jobs[0].Title)
	assert.Equal(t, "Imperial College London", got.Jobs[0].Employer)
	assert.Equal(t, "£59,808 to £71,793 per annum", got.Jobs[0].Salary)
	for _, j := range got.Jobs {
		assert.Equal(t, "London", j.Location)
	}
}

func TestSearchEmpty(t *testing.T) {
	c := newTestClient(t)
	got, err := c.Search(t.Context(), &SearchRequest{Keyword: "zzqqxxnotajob"})
	require.NoError(t, err)
	assert.Equal(t, 0, got.Total)
	assert.Equal(t, 0, got.TotalPages)
	assert.NotNil(t, got.Jobs)
	assert.Empty(t, got.Jobs)
}

func TestSearchURL(t *testing.T) {
	c := NewClient("https://www.jobs.ac.uk", nil)
	raw, err := c.searchURL(&SearchRequest{
		Keyword:                "technical manager",
		Locations:              []string{"Greater London", "london"},
		NonAcademicDisciplines: []string{"senior-management"},
		SalaryBands:            []string{"50000-59999", "60000-69999"},
		ContractTypes:          []string{"permanent"},
		Sort:                   SortClosingDate,
		Page:                   3,
	})
	require.NoError(t, err)
	u, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, "/search/", u.Path)
	q := u.Query()
	assert.Equal(t, "technical manager", q.Get("keywords"))
	assert.Equal(t, "2", q.Get("sortOrder"))
	assert.Equal(t, "25", q.Get("pageSize"))
	assert.Equal(t, "51", q.Get("startIndex"))
	assert.Equal(t, []string{"greater-london", "london"}, q["locationFacet[]"])
	assert.Equal(t, []string{"senior-management"}, q["nonAcademicDisciplineFacet[]"])
	assert.Equal(t, []string{"50000-59999", "60000-69999"}, q["salaryBandFacet[]"])
	assert.Equal(t, []string{"permanent"}, q["contractTypeFacet[]"])

	_, err = c.searchURL(&SearchRequest{Sort: "newest"})
	assert.Error(t, err)
}

func TestJobDetail(t *testing.T) {
	c := newTestClient(t)
	for _, in := range []string{"DTJ660", "dtj660", "https://www.jobs.ac.uk/job/DTJ660/technical-services-manager"} {
		got, err := c.JobDetail(t.Context(), in)
		require.NoError(t, err, in)
		assert.Equal(t, "DTJ660", got.ID)
		assert.Equal(t, "Technical Services Manager", got.Title)
		assert.True(t, strings.HasSuffix(got.URL, "/job/DTJ660/technical-services-manager"), got.URL)
		assert.Equal(t, "University for the Creative Arts", got.Employer)
		assert.Equal(t, "Farnham School of Creative and Cultural Industries: Technical", got.Department)
		assert.Equal(t, "Farnham", got.Location)
		assert.Equal(t, "£48,822 per annum", got.Salary)
		assert.Equal(t, "Full Time", got.Hours)
		assert.Equal(t, "Fixed-Term/Contract", got.ContractType)
		assert.Equal(t, "2026-10-23", got.PostedAt)
		assert.Equal(t, "2026-11-08", got.ClosesAt)
		assert.Equal(t, "26-AAAA1481W-0348-2", got.JobRef)
		assert.Equal(t, "https://jobs.ucreative.ac.uk/Vacancy.aspx?ref=26-AAAA1481W-0348-2", got.ApplyURL)
		assert.Contains(t, got.Description, "Technical Services Manager")
		assert.NotContains(t, got.Description, "<p>")
	}
}

func TestJobDetailNotFound(t *testing.T) {
	c := newTestClient(t)
	_, err := c.JobDetail(t.Context(), "ZZZ999")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestNormalizeJobID(t *testing.T) {
	for in, want := range map[string]string{
		"DTJ660":                          "DTJ660",
		" dtl650 ":                        "DTL650",
		"https://www.jobs.ac.uk/job/DTK759": "DTK759",
		"/job/DTJ824/technical-facilities-manager-maternity-cover?x=1": "DTJ824",
	} {
		got, err := NormalizeJobID(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, bad := range []string{"", "DT J660", "../etc", "https://www.jobs.ac.uk/search/"} {
		_, err := NormalizeJobID(bad)
		assert.Error(t, err, bad)
	}
}

func TestSlugify(t *testing.T) {
	assert.Equal(t, "greater-london", Slugify(" Greater London "))
	assert.Equal(t, "fixed-term-contract", Slugify("Fixed-Term/Contract"))
	assert.Equal(t, "50000-59999", Slugify("50000-59999"))
	assert.Equal(t, "", Slugify("  "))
}
