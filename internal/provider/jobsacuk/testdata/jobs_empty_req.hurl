# jobs.ac.uk search with no matches still returns 200 and a zero count.
GET https://www.jobs.ac.uk/search/?keywords=zzqqxxnotajob&sortOrder=0&pageSize=25&startIndex=1
User-Agent: openings-mcp/jobsacuk (personal job search; +https://github.com/amikai/openings-mcp)
Accept: text/html,application/xhtml+xml
HTTP 200
[Asserts]
body contains "job-count\">0<"
