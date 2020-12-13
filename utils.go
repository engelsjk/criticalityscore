package criticalityscore

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
)

func totalCount(resp *github.Response) int {

	links := parseLinkHeader(resp.Header)

	lastURL, ok := links["last"]
	if !ok {
		return 0
	}

	u, err := url.Parse(lastURL)
	if err != nil {
		return 0
	}

	m, _ := url.ParseQuery(u.RawQuery)

	pageCount, err := strconv.Atoi(m.Get("page"))
	if err != nil {
		return 0
	}

	return pageCount
}

func parseRepoURL(s string) (string, string) {
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", ""
	}

	if !(u.Host == "github.com") {
		return "", ""
	}

	p := strings.Split(u.Path, "/")

	if len(p) < 3 {
		return "", ""
	}

	return p[1], p[2]
}

func parseLinkHeader(header http.Header) map[string]string {
	links := make(map[string]string)
	linkHeaders := strings.Split(header.Get("link"), ", ")
	for _, linkHeader := range linkHeaders {
		lh := strings.Split(linkHeader, "; ")
		u := substr(lh[0], 1, -1)
		r := substr(lh[1], 5, -1)
		links[r] = u
	}
	return links
}

func pauseIfGitHubRateLimitExceeded(client *github.Client, ctx context.Context) {
	rateLimits, resp, err := client.RateLimits(ctx)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if rateLimits.Core.Remaining < 50 {
		log.Printf("rate limit exceeded, sleeping for an hour before retry.\n")
		time.Sleep(60 * time.Minute)
	}
}

func filterOrgName(orgName string) string {
	name := strings.ToLower(orgName)
	replacer := strings.NewReplacer("inc.", "", "llc", "", "@", "", " ", "")
	name = replacer.Replace(name)
	name = strings.TrimRight(name, ",")
	return name
}

func substr(input string, start int, end int) string {
	asRunes := []rune(input)
	if start >= len(asRunes) {
		return ""
	}
	if end > len(asRunes) {
		end = len(asRunes) - 1
	}
	if end <= 0 {
		end = len(asRunes) + end
	}
	return string(asRunes[start:end])
}
