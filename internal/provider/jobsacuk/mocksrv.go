package jobsacuk

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
	"strings"
)

//go:embed testdata/jobs_rsp.html
var mockJobsRsp []byte

//go:embed testdata/jobs_filtered_rsp.html
var mockJobsFilteredRsp []byte

//go:embed testdata/jobs_empty_rsp.html
var mockJobsEmptyRsp []byte

//go:embed testdata/job_detail_rsp.html
var mockJobDetailRsp []byte

//go:embed testdata/job_detail_notfound_rsp.html
var mockJobDetailNotFoundRsp []byte

// MockJobID is the advert id served by the detail fixture.
const MockJobID = "DTJ660"

// NewMockServer returns an httptest.Server that replays captured jobs.ac.uk
// HTML so tests never hit the live site. The caller must Close it.
func NewMockServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/search/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		q := r.URL.Query()
		switch {
		case q.Get("keywords") == "zzqqxxnotajob":
			w.Write(mockJobsEmptyRsp)
		case q.Get("locationFacet[]") != "":
			// Mirrors jobs_filtered_req.hurl (London facet, page 2).
			w.Write(mockJobsFilteredRsp)
		default:
			w.Write(mockJobsRsp)
		}
	})
	mux.HandleFunc("/job/", func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/job/")
		id, slug, _ := strings.Cut(rest, "/")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if id != MockJobID {
			w.WriteHeader(http.StatusNotFound)
			w.Write(mockJobDetailNotFoundRsp)
			return
		}
		// The live site redirects /job/{id} to /job/{id}/{slug}.
		if slug == "" {
			http.Redirect(w, r, "/job/"+MockJobID+"/technical-services-manager", http.StatusMovedPermanently)
			return
		}
		w.Write(mockJobDetailRsp)
	})
	return httptest.NewServer(mux)
}
