package openingsmcp

import (
	"context"

	"github.com/amikai/openings-mcp/internal/provider/jobsacuk"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var jobsacukSearchInputRawSchema = []byte(`{
	"type": "object",
	"properties": {
		"keyword": {
			"type": "string",
			"description": "Free-text search: role titles, skills, subjects, or employer names.",
			"minLength": 1
		},
		"locations": {
			"type": "array",
			"description": "Location filter as jobs.ac.uk location slugs, ORed together, e.g. london, greater-london, england, scotland, wales, northern-ireland, united-kingdom, west-midlands, or a city such as manchester. An unknown slug returns zero results rather than an error. Free text is slugified (\"Greater London\" becomes greater-london). Use this rather than putting a place in keyword: the site ignores free-text locations.",
			"items": {"type": "string", "minLength": 1}
		},
		"academic_disciplines": {
			"type": "array",
			"description": "Academic subject slugs, ORed: agriculture-food-and-veterinary, architecture-building-and-planning, biological-sciences, business-and-management-studies, computer-sciences, creative-arts-and-design, economics, education-studies-inc-tefl, engineering-and-technology, health-and-medical, historical-and-philosophical-studies, information-management-and-librarianship, languages-literature-and-culture, law, mathematics-and-statistics, media-and-communications, physical-and-environmental-sciences, politics-and-government, psychology, social-sciences-and-social-care, sport-and-leisure.",
			"items": {"type": "string", "minLength": 1}
		},
		"non_academic_disciplines": {
			"type": "array",
			"description": "Professional services area slugs, ORed: administrative, estates-and-facilities-management, finance-and-procurement, fundraising-alumni-bids-and-grants, health-wellbeing-and-care, hospitality-retail-conferences-and-events, human-resources, international-activities, it-services, laboratory-clinical-and-technician, legal-compliance-and-policy, library-services-data-and-information-management, pr-marketing-sales-and-communication, project-management-and-consulting, senior-management, sports-and-leisure, student-services, sustainability, web-design-and-development, other.",
			"items": {"type": "string", "minLength": 1}
		},
		"salary_bands": {
			"type": "array",
			"description": "Salary band slugs, ORed: up-to-9999, 10000-14999, 15000-19999, 20000-24999, 25000-29999, 30000-39999, 40000-49999, 50000-59999, 60000-69999, 70000-79999, 80000-89999, 90000-99999, over-100000. For a minimum salary, list every band from that floor upwards.",
			"items": {"type": "string", "minLength": 1}
		},
		"hours": {
			"type": "array",
			"description": "Hours: full-time and/or part-time.",
			"items": {"type": "string", "enum": ["full-time", "part-time"]}
		},
		"contract_types": {
			"type": "array",
			"description": "Contract types: permanent, fixed-term-contract, temporary.",
			"items": {"type": "string", "enum": ["permanent", "fixed-term-contract", "temporary"]}
		},
		"workplaces": {
			"type": "array",
			"description": "Workplace types: on-site, hybrid, remote.",
			"items": {"type": "string", "enum": ["on-site", "hybrid", "remote"]}
		},
		"employer_sectors": {
			"type": "array",
			"description": "Employer sectors, ORed: universities-higher-education-institutions, further-education, science-and-research-organisations, public-sector, charity-not-for-profit, commercial-organisations.",
			"items": {"type": "string", "minLength": 1}
		},
		"sort": {
			"type": "string",
			"description": "Result order. Defaults to relevance.",
			"enum": ["relevance", "date_placed", "closing_date"],
			"default": "relevance"
		},
		"page": {
			"type": "integer",
			"description": "1-based page number; 25 results per page.",
			"minimum": 1,
			"default": 1
		}
	},
	"additionalProperties": false
}`)

var jobsacukSearchInputSchema = mustSchema(jobsacukSearchInputRawSchema)

type jobsacukSearchInput struct {
	Keyword                string   `json:"keyword,omitempty"`
	Locations              []string `json:"locations,omitempty"`
	AcademicDisciplines    []string `json:"academic_disciplines,omitempty"`
	NonAcademicDisciplines []string `json:"non_academic_disciplines,omitempty"`
	SalaryBands            []string `json:"salary_bands,omitempty"`
	Hours                  []string `json:"hours,omitempty"`
	ContractTypes          []string `json:"contract_types,omitempty"`
	Workplaces             []string `json:"workplaces,omitempty"`
	EmployerSectors        []string `json:"employer_sectors,omitempty"`
	Sort                   string   `json:"sort,omitempty"`
	Page                   int      `json:"page,omitempty"`
}

type jobsacukSearchOutput struct {
	Total      int                   `json:"total" jsonschema:"Total number of matching jobs."`
	Page       int                   `json:"page" jsonschema:"Current 1-based page."`
	TotalPages int                   `json:"total_pages" jsonschema:"Total pages at 25 results per page."`
	Jobs       []jobsacuk.JobSummary `json:"jobs" jsonschema:"Matching jobs. id is the advert id for jobsacuk_get_job_detail. date_placed and closes are shown exactly as on the site, without a year (e.g. 23 Oct); the detail tool gives full ISO dates."`
}

func jobsacukMCPToHTTPRequest(in *jobsacukSearchInput) *jobsacuk.SearchRequest {
	return &jobsacuk.SearchRequest{
		Keyword:                in.Keyword,
		Locations:              in.Locations,
		AcademicDisciplines:    in.AcademicDisciplines,
		NonAcademicDisciplines: in.NonAcademicDisciplines,
		SalaryBands:            in.SalaryBands,
		HoursTypes:             in.Hours,
		ContractTypes:          in.ContractTypes,
		Workplaces:             in.Workplaces,
		EmployerSectors:        in.EmployerSectors,
		Sort:                   in.Sort,
		Page:                   in.Page,
	}
}

type jobsacukDetailInput struct {
	ID string `json:"id" jsonschema:"Advert id from jobsacuk_search_jobs (e.g. DTJ660), or a jobs.ac.uk job page URL."`
}

// RegisterJobsacuk registers jobs.ac.uk search and detail tools.
func RegisterJobsacuk(s *mcp.Server, c *jobsacuk.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "jobsacuk_search_jobs",
		Description: "Search jobs on jobs.ac.uk, the UK board for higher education, research, and university professional services roles (academic, technical, IT, estates, management).",
		Annotations: &mcp.ToolAnnotations{Title: "Search jobs.ac.uk jobs", ReadOnlyHint: true},
		InputSchema: jobsacukSearchInputSchema,
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in *jobsacukSearchInput) (*mcp.CallToolResult, *jobsacukSearchOutput, error) {
		res, err := c.Search(ctx, jobsacukMCPToHTTPRequest(in))
		if err != nil {
			return errorResult(err), nil, nil
		}
		return nil, &jobsacukSearchOutput{
			Total:      res.Total,
			Page:       res.Page,
			TotalPages: res.TotalPages,
			Jobs:       res.Jobs,
		}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "jobsacuk_get_job_detail",
		Description: "Get the full advert for a jobs.ac.uk job (pass id from jobsacuk_search_jobs): description, salary, hours, contract type, posted and closing dates, job reference, and the employer's apply link.",
		Annotations: &mcp.ToolAnnotations{Title: "Get jobs.ac.uk job details", ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in *jobsacukDetailInput) (*mcp.CallToolResult, *jobsacuk.JobDetail, error) {
		res, err := c.JobDetail(ctx, in.ID)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return nil, res, nil
	})
}
