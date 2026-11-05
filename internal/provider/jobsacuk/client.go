package jobsacuk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	searchPath = "/search/"
	detailPath = "/job/"
	// PageSize is the site's fixed search page size; larger pageSize values
	// are ignored upstream.
	PageSize = 25
	// UserAgent identifies the client honestly instead of posing as a browser.
	UserAgent = "openings-mcp/jobsacuk (personal job search; +https://github.com/amikai/openings-mcp)"
)

// Sort orders accepted by Search, mapped to the site's sortOrder values.
const (
	SortRelevance   = "relevance"
	SortDatePlaced  = "date_placed"
	SortClosingDate = "closing_date"
)

var sortOrders = map[string]string{
	SortRelevance:   "0",
	SortDatePlaced:  "1",
	SortClosingDate: "2",
}

// ErrNotFound is returned when a job id does not exist or has been removed.
var ErrNotFound = errors.New("jobsacuk: job not found")

// jobIDRE matches jobs.ac.uk advert ids such as DTJ660.
var jobIDRE = regexp.MustCompile(`^[A-Za-z0-9]{3,12}$`)

// jobURLRE pulls the id out of a /job/{id}[/{slug}] path.
var jobURLRE = regexp.MustCompile(`/job/([A-Za-z0-9]+)(?:[/?#]|$)`)

// Client talks to jobs.ac.uk's public HTML pages.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient builds a Client. When httpClient is nil, http.DefaultClient is used.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient, baseURL: strings.TrimRight(baseURL, "/")}
}

// SearchRequest is a jobs.ac.uk search. Facet fields take the site's slugs;
// each may hold several values, which the site ORs together.
type SearchRequest struct {
	Keyword string
	// Locations are locationFacet slugs, e.g. "london", "greater-london",
	// "scotland". Free text such as "Greater London" is slugified.
	Locations []string
	// AcademicDisciplines are academicDisciplineFacet slugs, e.g.
	// "computer-sciences", "creative-arts-and-design".
	AcademicDisciplines []string
	// NonAcademicDisciplines are nonAcademicDisciplineFacet slugs, e.g.
	// "it-services", "senior-management", "laboratory-clinical-and-technician".
	NonAcademicDisciplines []string
	// SalaryBands are salaryBandFacet slugs, e.g. "50000-59999", "over-100000".
	SalaryBands []string
	// HoursTypes are hoursTypeFacet slugs: "full-time", "part-time".
	HoursTypes []string
	// ContractTypes are contractTypeFacet slugs: "permanent",
	// "fixed-term-contract", "temporary".
	ContractTypes []string
	// Workplaces are workplaceFacet slugs: "on-site", "hybrid", "remote".
	Workplaces []string
	// EmployerSectors are employerSectorFacet slugs, e.g.
	// "universities-higher-education-institutions".
	EmployerSectors []string
	// Sort is SortRelevance (default), SortDatePlaced or SortClosingDate.
	Sort string
	// Page is 1-based. Zero means page 1.
	Page int
}

// SearchResponse is one page of search results.
type SearchResponse struct {
	Total      int          `json:"total"`
	Page       int          `json:"page"`
	TotalPages int          `json:"total_pages"`
	Jobs       []JobSummary `json:"jobs"`
}

// JobSummary is one search result card. Dates are the card's own text,
// which has no year (e.g. "23 Oct").
type JobSummary struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Employer   string `json:"employer,omitempty"`
	Department string `json:"department,omitempty"`
	Location   string `json:"location,omitempty"`
	Salary     string `json:"salary,omitempty"`
	DatePlaced string `json:"date_placed,omitempty"`
	Closes     string `json:"closes,omitempty"`
}

// JobDetail is a parsed job advert page.
type JobDetail struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	ApplyURL   string `json:"apply_url,omitempty"`
	Employer   string `json:"employer,omitempty"`
	Department string `json:"department,omitempty"`
	Location   string `json:"location,omitempty"`
	// Salary is the advert's salary text, e.g. "£48,822 per annum".
	Salary       string `json:"salary,omitempty"`
	Hours        string `json:"hours,omitempty"`
	ContractType string `json:"contract_type,omitempty"`
	// PostedAt and ClosesAt are ISO 8601 dates (YYYY-MM-DD) from JSON-LD.
	PostedAt    string `json:"posted_at,omitempty"`
	ClosesAt    string `json:"closes_at,omitempty"`
	JobRef      string `json:"job_ref,omitempty"`
	Description string `json:"description,omitempty"`
}

// Search runs a server-side search and parses one page of results.
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req == nil {
		req = &SearchRequest{}
	}
	u, err := c.searchURL(req)
	if err != nil {
		return nil, err
	}
	res, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	page := req.Page
	if page < 1 {
		page = 1
	}
	return parseSearchHTML(res.Body, c.baseURL, page)
}

// JobDetail fetches and parses /job/{id}. id may also be a job page URL.
func (c *Client) JobDetail(ctx context.Context, id string) (*JobDetail, error) {
	id, err := NormalizeJobID(id)
	if err != nil {
		return nil, err
	}
	res, err := c.get(ctx, c.baseURL+detailPath+url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return parseDetailHTML(res.Body, id, res.Request.URL.String())
}

// NormalizeJobID accepts a bare advert id (DTJ660) or a jobs.ac.uk job URL and
// returns the upper-cased id.
func NormalizeJobID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("job id is required")
	}
	if strings.Contains(raw, "/") {
		m := jobURLRE.FindStringSubmatch(raw)
		if m == nil {
			return "", fmt.Errorf("not a jobs.ac.uk job URL: %q", raw)
		}
		raw = m[1]
	}
	if !jobIDRE.MatchString(raw) {
		return "", fmt.Errorf("invalid job id %q (expected something like DTJ660)", raw)
	}
	return strings.ToUpper(raw), nil
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify turns free text such as "Greater London" into a facet slug
// ("greater-london"). Values that are already slugs pass through unchanged.
func Slugify(s string) string {
	return strings.Trim(slugRE.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-"), "-")
}

func (c *Client) searchURL(req *SearchRequest) (string, error) {
	sort := req.Sort
	if sort == "" {
		sort = SortRelevance
	}
	order, ok := sortOrders[sort]
	if !ok {
		return "", fmt.Errorf("invalid sort %q (want %s, %s or %s)", sort, SortRelevance, SortDatePlaced, SortClosingDate)
	}
	page := req.Page
	if page < 1 {
		page = 1
	}
	q := url.Values{}
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		q.Set("keywords", kw)
	}
	q.Set("sortOrder", order)
	q.Set("pageSize", strconv.Itoa(PageSize))
	q.Set("startIndex", strconv.Itoa((page-1)*PageSize+1))
	for _, f := range []struct {
		name   string
		values []string
	}{
		{"locationFacet", req.Locations},
		{"academicDisciplineFacet", req.AcademicDisciplines},
		{"nonAcademicDisciplineFacet", req.NonAcademicDisciplines},
		{"salaryBandFacet", req.SalaryBands},
		{"hoursTypeFacet", req.HoursTypes},
		{"contractTypeFacet", req.ContractTypes},
		{"workplaceFacet", req.Workplaces},
		{"employerSectorFacet", req.EmployerSectors},
	} {
		for _, v := range f.values {
			if s := Slugify(v); s != "" {
				q.Add(f.name+"[]", s)
			}
		}
	}
	return c.baseURL + searchPath + "?" + q.Encode(), nil
}

// get issues a GET and returns the response for the caller to read; the
// caller owns res.Body.
func (c *Client) get(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusNotFound {
		res.Body.Close()
		return nil, ErrNotFound
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		res.Body.Close()
		return nil, fmt.Errorf("jobsacuk: unexpected status %d for %s", res.StatusCode, rawURL)
	}
	return res, nil
}
