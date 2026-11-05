# jobs.ac.uk search filtered to the London location facet, date order, page 2.
GET https://www.jobs.ac.uk/search/?keywords=technical+manager&locationFacet%5B%5D=london&sortOrder=1&pageSize=25&startIndex=26
User-Agent: openings-mcp/jobsacuk (personal job search; +https://github.com/amikai/openings-mcp)
Accept: text/html,application/xhtml+xml
HTTP 200
[Asserts]
body contains "job-count"
body contains "Location:"
