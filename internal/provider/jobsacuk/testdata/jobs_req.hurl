# jobs.ac.uk search: keyword "technical manager", relevance order, page 1.
GET https://www.jobs.ac.uk/search/?keywords=technical+manager&sortOrder=0&pageSize=25&startIndex=1
User-Agent: openings-mcp/jobsacuk (personal job search; +https://github.com/amikai/openings-mcp)
Accept: text/html,application/xhtml+xml
HTTP 200
[Asserts]
body contains "job-count"
body contains "j-search-result__result"
