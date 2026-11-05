# jobs.ac.uk advert page; /job/{id} 301-redirects to /job/{id}/{slug}.
GET https://www.jobs.ac.uk/job/DTJ660/technical-services-manager
User-Agent: openings-mcp/jobsacuk (personal job search; +https://github.com/amikai/openings-mcp)
Accept: text/html,application/xhtml+xml
HTTP 200
[Asserts]
body contains "application/ld+json"
body contains "JobPosting"
body contains "job-description"
