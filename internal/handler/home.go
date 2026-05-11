package handler

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

type Format struct {
	Name       string
	Ecosystems []string
}

type HomeData struct {
	Formats          []Format
	AppName          string
	AppDescription   string
	MetaTitle        string
	MetaDescription  string
	GithubRepoName   string
	Services         map[string][]Service
	RequestBaseURL   string
}

type Service struct {
	Name string
	URL  string
}

type ErrorData struct {
	AppName          string
	AppDescription   string
	MetaTitle        string
	MetaDescription  string
	GithubRepoName   string
	Services         map[string][]Service
	RequestBaseURL   string
	StatusCode       int
	StatusText       string
}

var formats = []Format{
	{Name: ".tgz/.tar.gz", Ecosystems: []string{"npm", "pub", "cran", "hackage", "puppet"}},
	{Name: ".zip", Ecosystems: []string{"go", "elm"}},
	{Name: ".tar", Ecosystems: []string{"hex"}},
	{Name: ".gem", Ecosystems: []string{"rubygems"}},
	{Name: ".nupkg", Ecosystems: []string{"nuget"}},
	{Name: ".crate", Ecosystems: []string{"cargo"}},
}

var services = map[string][]Service{
	"Data": {
		{Name: "Packages", URL: "https://packages.ecosyste.ms"},
		{Name: "Repositories", URL: "https://repos.ecosyste.ms"},
		{Name: "Advisories", URL: "https://advisories.ecosyste.ms"},
	},
	"Tools": {
		{Name: "Dependency Parser", URL: "https://parser.ecosyste.ms"},
		{Name: "Dependency Resolver", URL: "https://resolve.ecosyste.ms"},
		{Name: "SBOM Parser", URL: "https://sbom.ecosyste.ms"},
		{Name: "License Parser", URL: "https://licenses.ecosyste.ms"},
		{Name: "Digest", URL: "https://digest.ecosyste.ms"},
		{Name: "Archives", URL: "https://archives.ecosyste.ms"},
		{Name: "Diff", URL: "https://diff.ecosyste.ms"},
		{Name: "Summary", URL: "https://summary.ecosyste.ms"},
	},
	"Indexes": {
		{Name: "Timeline", URL: "https://timeline.ecosyste.ms"},
		{Name: "Commits", URL: "https://commits.ecosyste.ms"},
		{Name: "Issues", URL: "https://issues.ecosyste.ms"},
		{Name: "Sponsors", URL: "https://sponsors.ecosyste.ms"},
		{Name: "Docker", URL: "https://docker.ecosyste.ms"},
		{Name: "Open Collective", URL: "https://opencollective.ecosyste.ms"},
		{Name: "Dependabot", URL: "https://dependabot.ecosyste.ms"},
	},
	"Applications": {
		{Name: "Funds", URL: "https://funds.ecosyste.ms"},
		{Name: "Dashboards", URL: "https://dashboards.ecosyste.ms"},
	},
	"Experiments": {
		{Name: "OST", URL: "https://ost.ecosyste.ms"},
		{Name: "Papers", URL: "https://papers.ecosyste.ms"},
		{Name: "Awesome", URL: "https://awesome.ecosyste.ms"},
		{Name: "Ruby", URL: "https://ruby.ecosyste.ms"},
	},
}

// ServiceCategories returns category names in display order.
var serviceCategories = []string{"Data", "Tools", "Indexes", "Applications", "Experiments"}

var templates *template.Template

func InitTemplates(templateDir string) error {
	funcMap := template.FuncMap{
		"join": func(items []string, sep string) string {
			result := ""
			for i, item := range items {
				if i > 0 {
					result += sep
				}
				result += item
			}
			return result
		},
		"serviceCategories": func() []string {
			return serviceCategories
		},
	}

	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob(filepath.Join(templateDir, "*.html"))
	return err
}

func githubRepoName() string {
	if name := os.Getenv("GITHUB_REPO_NAME"); name != "" {
		return name
	}
	return "archives"
}

func HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		HandleNotFound(w, r)
		return
	}

	data := HomeData{
		Formats:         formats,
		AppName:         "Archives",
		AppDescription:  "An open API service for inspecting package archives and files from many open source software ecosystems. Explore package contents without downloading.",
		MetaTitle:       "Ecosyste.ms: Archives",
		MetaDescription: "An open API service for inspecting package archives and files from many open source software ecosystems. Explore package contents without downloading.",
		GithubRepoName:  githubRepoName(),
		Services:        services,
		RequestBaseURL:  requestBaseURL(r),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	templates.ExecuteTemplate(w, "layout.html", data)
}

func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if isJSONRequest(accept) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	data := ErrorData{
		AppName:        "Archives",
		AppDescription: "An open API service for inspecting package archives and files from many open source software ecosystems. Explore package contents without downloading.",
		MetaTitle:      "Not Found | Ecosyste.ms: Archives",
		GithubRepoName: githubRepoName(),
		Services:       services,
		RequestBaseURL: requestBaseURL(r),
		StatusCode:     404,
		StatusText:     "Not Found",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	templates.ExecuteTemplate(w, "error.html", data)
}

func HandleUnprocessable(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if isJSONRequest(accept) {
		writeError(w, http.StatusUnprocessableEntity, "unprocessable")
		return
	}

	data := ErrorData{
		AppName:        "Archives",
		AppDescription: "An open API service for inspecting package archives and files from many open source software ecosystems. Explore package contents without downloading.",
		MetaTitle:      "Unprocessable | Ecosyste.ms: Archives",
		GithubRepoName: githubRepoName(),
		Services:       services,
		RequestBaseURL: requestBaseURL(r),
		StatusCode:     422,
		StatusText:     "Unprocessable Entity",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	templates.ExecuteTemplate(w, "error.html", data)
}

func HandleInternalError(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if isJSONRequest(accept) {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	data := ErrorData{
		AppName:        "Archives",
		AppDescription: "An open API service for inspecting package archives and files from many open source software ecosystems. Explore package contents without downloading.",
		MetaTitle:      "Internal Server Error | Ecosyste.ms: Archives",
		GithubRepoName: githubRepoName(),
		Services:       services,
		RequestBaseURL: requestBaseURL(r),
		StatusCode:     500,
		StatusText:     "Internal Server Error",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	templates.ExecuteTemplate(w, "error.html", data)
}

func isJSONRequest(accept string) bool {
	return accept == "application/json" || accept == "text/json"
}

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd == "https" || fwd == "http" {
		scheme = fwd
	}
	return scheme + "://" + r.Host
}
