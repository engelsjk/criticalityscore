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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	ErrRepoNotProvided                error = fmt.Errorf("please provided a repo url")
	ErrInvalidGitHubURL               error = fmt.Errorf("invalid github url")
	ErrRepoNotFound                   error = fmt.Errorf("repo not found")
	ErrAPIResponseError               error = fmt.Errorf("github api response error, please try again")
	ErrCommitFrequencyBeingCalculated error = fmt.Errorf("commit frequency is being calculated by github, please try again")
)

// GitHubRepository is an object that provides a GitHub client interface for a single repository.
type GitHubRepository struct {
	ctx    context.Context
	client *github.Client
	R      *github.Repository
	Error  error
}

// LoadRepository returns a GitHubRepository object from a GitHub repository URL
// and an authorized GitHUB personal access token.
func LoadRepository(repoURL, token string) (GitHubRepository, error) {

	if repoURL == "" {
		fmt.Println(ErrRepoNotProvided.Error())
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	pauseIfGitHubRateLimitExceeded(client, ctx)

	owner, name := parseRepoURL(repoURL)

	if owner == "" || name == "" {
		return GitHubRepository{}, ErrInvalidGitHubURL
	}

	r, _, err := client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return GitHubRepository{}, ErrRepoNotFound
	}

	return GitHubRepository{
		ctx:    ctx,
		client: client,
		R:      r,
	}, nil
}

// Criteria important for ranking.

// CreatedSince returns the number of months since the repository was created.
func (ghr GitHubRepository) CreatedSince() int {
	difference := time.Since(ghr.R.CreatedAt.Time)
	return int(math.Round(difference.Hours() / 24.0 / 30.0))
}

// UpdatedSince returns the number of months since the last commit.
func (ghr GitHubRepository) UpdatedSince() int {

	commits, _, err := ghr.client.Repositories.ListCommits(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), nil)
	if err != nil {
		ghr.Error = err
		return 0
	}

	lastCommit := commits[0]
	difference := time.Since(lastCommit.Commit.Author.GetDate())
	return int(math.Round(difference.Hours() / 24.0 / 30.0))
}

// Contributors returns the number of all contributors.
func (ghr GitHubRepository) Contributors() int {

	opts := &github.ListContributorsOptions{
		Anon: "true",
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	_, resp, err := ghr.client.Repositories.ListContributors(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), opts)
	if err != nil {
		ghr.Error = err
		return 0
	}

	return totalCount(resp)
}

// ContributorOrgs returns a map of companies associated with each of the top contributors.
func (ghr GitHubRepository) ContributorOrgs() map[string]bool {

	opts := &github.ListContributorsOptions{
		Anon: "false",
		ListOptions: github.ListOptions{
			PerPage: 25,
		},
	}
	var allContributors []*github.Contributor
	for {
		contributors, resp, err := ghr.client.Repositories.ListContributors(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), opts)
		if err != nil {
			ghr.Error = err
			return nil
		}
		allContributors = append(allContributors, contributors...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage

		if len(allContributors) > TopContributorCount {
			break
		}
	}

	orgs := make(map[string]bool)

	if len(allContributors) > 5000 {
		for i := 0; i < 10; i++ {
			orgs[string(i)] = true
		}
		return orgs
	}

	var allUsers []*github.User
	for _, contributor := range allContributors[:TopContributorCount] {
		user, _, err := ghr.client.Users.GetByID(ghr.ctx, contributor.GetID())
		if err != nil {
			continue
		}

		allUsers = append(allUsers, user)

		company := user.GetCompany()
		if company == "" {
			continue
		}
		name := filterOrgName(company)
		orgs[name] = true
	}

	return orgs
}

// CommitFrequency returns the weekly average number of commits.
func (ghr GitHubRepository) CommitFrequency() float64 {

	weekStats, resp, err := ghr.client.Repositories.ListCommitActivity(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName())
	if err != nil {
		if resp.StatusCode == 202 {
			ghr.Error = ErrCommitFrequencyBeingCalculated
			return 0
		}
		ghr.Error = err
		return 0
	}

	total := 0
	for _, weekStat := range weekStats {
		total += weekStat.GetTotal()
	}

	return math.Round(float64(total)/52.0*10.0) / 10
}

// RecentReleases returns the number of recent repository releases.
// If none found within the number of ReleaseLookbackDays, then an estimate
// is calculated based on totalTags / daysSinceCreation * ReleaseLookbackDays.
func (ghr GitHubRepository) RecentReleases() int {

	opts := &github.ListOptions{
		PerPage: 100,
	}
	var allReleases []*github.RepositoryRelease
	for {
		releases, resp, err := ghr.client.Repositories.ListReleases(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), opts)
		if err != nil {
			ghr.Error = err
			return 0
		}
		allReleases = append(allReleases, releases...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	total := 0
	for _, release := range allReleases {
		if time.Since(release.CreatedAt.Time).Hours()/24.0 > ReleaseLookbackDays {
			continue
		}
		total++
	}

	if total == 0 {
		daysSinceCreation := int(time.Since(ghr.R.CreatedAt.Time) / 24.0)
		if daysSinceCreation == 0 {
			return 0
		}

		opts := &github.ListOptions{
			PerPage: 1,
		}
		_, resp2, err := ghr.client.Repositories.ListTags(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), opts)
		if err != nil {
			ghr.Error = err
			return 0
		}
		totalTags := totalCount(resp2)

		total = totalTags / daysSinceCreation * ReleaseLookbackDays
	}
	return total
}

// UpdatedIssues returns the number of all repository issues.
func (ghr GitHubRepository) UpdatedIssues() int {

	issuesSinceTime := time.Now().Add(-IssueLookbackDays * 24.0 * time.Hour)
	opts := &github.IssueListByRepoOptions{
		State: "all",
		Since: issuesSinceTime,
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	_, resp, err := ghr.client.Issues.ListByRepo(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), opts)
	if err != nil {
		ghr.Error = err
		return 0
	}

	return totalCount(resp)
}

// ClosedIssues returns the number of closed repository issues.
func (ghr GitHubRepository) ClosedIssues() int {

	issuesSinceTime := time.Now().Add(-IssueLookbackDays * 24.0 * time.Hour)
	opts := &github.IssueListByRepoOptions{
		State: "closed",
		Since: issuesSinceTime,
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	_, resp, err := ghr.client.Issues.ListByRepo(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), opts)
	if err != nil {
		ghr.Error = err
		return 0
	}

	return totalCount(resp)
}

// CommentFrequency returns the ratio of comments to issues.
func (ghr GitHubRepository) CommentFrequency(issueCount int) float64 {

	if issueCount == 0 {
		return 0
	}

	issuesSinceTime := time.Now().Add(-IssueLookbackDays * 24.0 * time.Hour)
	opts := &github.IssueListCommentsOptions{
		Since: issuesSinceTime,
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	}

	_, resp, err := ghr.client.Issues.ListComments(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), 0, opts)
	if err != nil {
		ghr.Error = err
		return 0
	}

	commentCount := totalCount(resp)

	return math.Round(float64(commentCount)/float64(issueCount)*10) / 10
}

// Dependents returns the number of search results that contain the repository name as in a commit.
func (ghr GitHubRepository) Dependents() int {

	params := url.Values{}
	params.Add("q", fmt.Sprintf(`"%s/%s"`, ghr.R.GetOwner().GetLogin(), ghr.R.GetName()))
	params.Add("type", "commits")

	dependentsURL := fmt.Sprintf(`https://github.com/search?%s`, params.Encode())

	var content []byte
	for i := 1; i <= 3; i++ {
		resp, err := http.Get(dependentsURL)
		if err != nil {
			continue
		}
		if resp.StatusCode == 200 {
			content, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			break
		}
		time.Sleep(10 * time.Second)
	}

	match := DependentsRegex.FindSubmatch(content)

	if len(match) == 0 {
		return 0
	}

	b := bytes.ReplaceAll(match[1], []byte(","), []byte(""))
	b = bytes.TrimSpace(b)
	dependentsCount, _ := strconv.Atoi(string(b))
	return dependentsCount
}

// func Paginate() {
// 	issuesSinceTime := time.Now().Add(-IssueLookbackDays * 24.0 * time.Hour)
// 	opts := &github.IssueListCommentsOptions{
// 		Since: issuesSinceTime,
// 		ListOptions: github.ListOptions{
// 			PerPage: 100,
// 		},
// 	}
// 	var allComments []*github.IssueComment
// 	for {
// 		comments, resp, err := ghr.client.Issues.ListComments(ghr.ctx, ghr.R.GetOwner().GetLogin(), ghr.R.GetName(), 0, opts)
// 		if err != nil {
// 			panic(err)
// 		}
// 		allComments = append(allComments, comments...)
// 		if resp.NextPage == 0 {
// 			resp.Body.Close()
// 			break
// 		}
// 		opts.Page = resp.NextPage
// 		resp.Body.Close()
// 	}
// }
