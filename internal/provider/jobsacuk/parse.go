package jobsacuk

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/jaytaylor/html2text"
)

// parseSearchHTML reads a /search/ page. baseURL resolves relative job links.
func parseSearchHTML(r io.Reader, baseURL string, page int) (*SearchResponse, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse search HTML: %w", err)
	}
	countSel := doc.Find("strong.job-count").First()
	if countSel.Length() == 0 {
		return nil, fmt.Errorf("jobsacuk: search page has no job count; the page layout may have changed")
	}
	total, err := strconv.Atoi(strings.ReplaceAll(clean(countSel.Text()), ",", ""))
	if err != nil {
		return nil, fmt.Errorf("jobsacuk: unreadable job count %q", clean(countSel.Text()))
	}

	out := &SearchResponse{
		Total:      total,
		Page:       page,
		TotalPages: (total + PageSize - 1) / PageSize,
		Jobs:       []JobSummary{},
	}
	doc.Find("div.j-search-result__result").Each(func(_ int, card *goquery.Selection) {
		text := card.Find("div.j-search-result__text").First()
		link := text.ChildrenFiltered("a").First()
		href, _ := link.Attr("href")
		m := jobURLRE.FindStringSubmatch(href)
		if m == nil {
			return
		}
		job := JobSummary{
			ID:         strings.ToUpper(m[1]),
			Title:      clean(link.Text()),
			URL:        absURL(baseURL, href),
			Department: clean(text.Find(".j-search-result__department").Text()),
			Employer:   clean(text.Find(".j-search-result__employer").Text()),
			Salary:     strings.TrimSpace(strings.TrimPrefix(clean(text.Find(".j-search-result__info").Text()), "Salary:")),
			Closes:     clean(card.Find(".j-search-result__date--blue").First().Text()),
		}
		text.ChildrenFiltered("div").Each(func(_ int, d *goquery.Selection) {
			t := clean(d.Text())
			switch {
			case strings.HasPrefix(t, "Location:"):
				job.Location = strings.TrimSpace(strings.TrimPrefix(t, "Location:"))
			case strings.HasPrefix(t, "Date Placed:"):
				job.DatePlaced = strings.TrimSpace(strings.TrimPrefix(t, "Date Placed:"))
			}
		})
		out.Jobs = append(out.Jobs, job)
	})
	return out, nil
}

// jobPosting is the subset of the advert's schema.org JSON-LD we read.
type jobPosting struct {
	Type               string `json:"@type"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	DatePosted         string `json:"datePosted"`
	ValidThrough       string `json:"validThrough"`
	HiringOrganization struct {
		Name       string `json:"name"`
		Department struct {
			Name string `json:"name"`
		} `json:"department"`
	} `json:"hiringOrganization"`
	JobLocation []struct {
		Address struct {
			AddressLocality string `json:"addressLocality"`
		} `json:"address"`
	} `json:"jobLocation"`
}

// parseDetailHTML reads a /job/{id}/{slug} page. pageURL is the final URL
// after redirects.
func parseDetailHTML(r io.Reader, id, pageURL string) (*JobDetail, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse detail HTML: %w", err)
	}
	var lp jobPosting
	doc.Find(`script[type="application/ld+json"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		var candidate jobPosting
		if json.Unmarshal([]byte(s.Text()), &candidate) == nil && candidate.Type == "JobPosting" {
			lp = candidate
			return false
		}
		return true
	})

	d := &JobDetail{
		ID:         id,
		URL:        pageURL,
		Title:      firstNonEmpty(lp.Title, clean(doc.Find("h1.j-advert__title").First().Text())),
		Employer:   lp.HiringOrganization.Name,
		Department: lp.HiringOrganization.Department.Name,
		PostedAt:   isoDate(lp.DatePosted),
		ClosesAt:   isoDate(lp.ValidThrough),
	}
	if d.Title == "" {
		return nil, fmt.Errorf("jobsacuk: job %s page has no title; the page layout may have changed", id)
	}
	if len(lp.JobLocation) > 0 {
		d.Location = lp.JobLocation[0].Address.AddressLocality
	}

	doc.Find(".j-advert-details__container tr").Each(func(_ int, row *goquery.Selection) {
		key := strings.TrimSuffix(clean(row.Find("th").Text()), ":")
		val := clean(row.Find("td").Text())
		switch key {
		case "Location":
			// The table's text is what candidates see; prefer it when present.
			if val != "" {
				d.Location = val
			}
		case "Salary":
			d.Salary = val
		case "Hours":
			d.Hours = val
		case "Contract Type":
			d.ContractType = val
		case "Job Ref":
			d.JobRef = val
		}
	})

	if href, ok := doc.Find("#job-apply-button").Closest("a").Attr("href"); ok {
		d.ApplyURL = strings.TrimSpace(href)
	}

	descHTML, _ := doc.Find("#job-description").First().Html()
	if strings.TrimSpace(descHTML) == "" {
		descHTML = lp.Description
	}
	if descHTML != "" {
		if text, err := html2text.FromString(descHTML, html2text.Options{}); err == nil {
			d.Description = strings.TrimSpace(text)
		}
	}
	return d, nil
}

// clean collapses runs of whitespace to single spaces.
func clean(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// isoDate trims a JSON-LD datetime such as 2026-10-23T00:00:00+00:00 to its
// YYYY-MM-DD date.
func isoDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

func absURL(baseURL, href string) string {
	u, err := url.Parse(href)
	if err != nil || u.IsAbs() {
		return href
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}
	return base.ResolveReference(u).String()
}
