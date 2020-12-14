// # Copyright 2020 Jon Engelsman
// # Copyright 2020 Google LLC
// #
// # Licensed under the Apache License, Version 2.0 (the "License");
// # you may not use this file except in compliance with the License.
// # You may obtain a copy of the License at
// #
// #      http://www.apache.org/licenses/LICENSE-2.0
// #
// # Unless required by applicable law or agreed to in writing, software
// # distributed under the License is distributed on an "AS IS" BASIS,
// # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// # See the License for the specific language governing permissions and
// # limitations under the License.

package criticalityscore

import (
	"context"
	"fmt"
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

func parseAdditionalParams(params []string) ([]AdditionalParam, error) {
	if len(params) == 0 {
		return nil, nil
	}
	additionalParams := []AdditionalParam{}
	for _, p := range params {
		ps := strings.Split(p, ":")
		if len(ps) != 3 {
			return nil, fmt.Errorf("param string should have 3 values (value:weight:threshold)")
		}

		v, err := strconv.ParseFloat(ps[0], 64)
		if err != nil {
			return nil, fmt.Errorf("param value should be type float64")
		}
		w, err := strconv.ParseFloat(ps[1], 64)
		if err != nil {
			return nil, fmt.Errorf("param weight should be type float64")
		}
		mt, err := strconv.ParseFloat(ps[2], 64)
		if err != nil {
			return nil, fmt.Errorf("param max_threshold should be type float64")
		}

		param := AdditionalParam{
			Value:        v,
			Weight:       w,
			MaxThreshold: mt,
		}

		additionalParams = append(additionalParams, param)
	}

	if len(additionalParams) == 0 {
		return nil, nil
	}

	return additionalParams, nil
}

func parseLinkHeader(header http.Header) map[string]string {
	links := make(map[string]string)
	linkHeaders := strings.Split(header.Get("link"), ", ")
	for _, linkHeader := range linkHeaders {
		lh := strings.Split(linkHeader, "; ")
		if len(lh) < 2 {
			continue
		}
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
		waitTime := rateLimits.Core.Reset.Sub(time.Now())
		log.Printf("rate limit exceeded, sleeping for %0.0f seconds before retry.\n", waitTime.Seconds())
		time.Sleep(waitTime * time.Second)
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
