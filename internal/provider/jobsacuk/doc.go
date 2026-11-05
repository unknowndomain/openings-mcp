// Package jobsacuk searches public job listings on jobs.ac.uk, the UK job
// board for higher education, research and related sectors (run by Warwick
// University Services Limited).
//
// There is no public API, and the subject-area RSS feeds the site used to
// publish (/feeds/subject-areas/...) were retired in a site relaunch and now
// return 404. Both operations therefore read server-rendered HTML:
//
//   - Search is GET /search/ with keywords, sortOrder, pageSize and a
//     1-based startIndex, plus optional facet filters sent as repeated
//     "<facet>[]" query parameters (locationFacet[], salaryBandFacet[], ...).
//     Each result is a div.j-search-result__result card. The page size is
//     fixed at 25: larger pageSize values are ignored by the site.
//   - Detail is GET /job/{id}, which 301-redirects to /job/{id}/{slug}. The
//     page carries a schema.org JobPosting in JSON-LD (dates, salary,
//     employer, location), which is preferred over the visible layout, plus a
//     details table (hours, contract type, job reference) and the
//     #job-description body.
//
// Quirks:
//
//   - The free-text "location" search parameter is ignored unless the site's
//     own Google Places autocomplete has supplied a placeId, and combining it
//     with "distance" returns HTTP 500. Location filtering therefore uses the
//     locationFacet[] slug (e.g. "london", "greater-london", "scotland").
//   - Search cards show "Date Placed" and "Closes" without a year (for
//     example "23 Oct"). They are returned exactly as shown; full ISO dates
//     are only available from the detail page.
//   - Unknown facet slugs are not an error upstream; the site just returns
//     zero results.
//
// Terms of use allow copying site material for personal use only, so this
// provider is intended for personal, low-frequency searches. It identifies
// itself with an honest User-Agent rather than a browser one.
package jobsacuk
