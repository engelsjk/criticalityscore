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

// todo: add additional param parse/validator from value:weight:threshold args

package criticalityscore

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
)

type Score struct {
	Name                string  `json:"name"`
	URL                 string  `json:"url"`
	Language            string  `json:"language"`
	CreatedSince        int     `json:"created_since"`
	UpdatedSince        int     `json:"updated_since"`
	ContributorCount    int     `json:"contributor_count"`
	OrgCount            int     `json:"org_count"`
	CommitFrequency     float64 `json:"commit_frequency"`
	RecentReleasesCount int     `json:"recent_release_count"`
	ClosedIssuesCount   int     `json:"closed_issues_count"`
	UpdatedIssuesCount  int     `json:"updated_issues_count"`
	CommentFrequency    float64 `json:"comment_frequency"`
	DependentsCount     int     `json:"dependents_count"`
	CriticalityScore    float64 `json:"criticality_score"`
}

func ParamScore(param interface{}, maxValue, weight float64) float64 {
	var p float64
	switch v := param.(type) {
	case float64:
		p = v
	case int:
		p = float64(v)
	}
	return math.Log(1.0+p) / math.Log(1.0+math.Max(p, maxValue)) * weight
}

type AdditionalParam struct {
	Value        float64
	Weight       float64
	MaxThreshold float64
}

func RepositoryStats(ghr GitHubRepository, additionalParams []AdditionalParam) (Score, error) {

	additionalParamsTotalWeight := 0.0
	additionalParamsScore := 0.0

	for _, param := range additionalParams {
		additionalParamsTotalWeight += param.Weight
		additionalParamsScore += ParamScore(param.Value, param.MaxThreshold, param.Weight)
	}

	score := Score{
		Name:     fmt.Sprintf("%s/%s", ghr.R.GetOwner().GetLogin(), ghr.R.GetName()),
		URL:      ghr.R.GetHTMLURL(),
		Language: ghr.R.GetLanguage(),
	}

	wg := new(sync.WaitGroup)
	wg.Add(9)

	go func() {
		score.CreatedSince = ghr.CreatedSince()
		wg.Done()
	}()

	go func() {
		score.UpdatedSince = ghr.UpdatedSince()
		wg.Done()
	}()

	go func() {
		score.ContributorCount = ghr.Contributors()
		wg.Done()
	}()

	go func() {
		score.OrgCount = len(ghr.ContributorOrgs())
		wg.Done()
	}()

	go func() {
		score.CommitFrequency = ghr.CommitFrequency()
		wg.Done()
	}()

	go func() {
		score.RecentReleasesCount = ghr.RecentReleases()
		wg.Done()
	}()

	go func() {
		score.ClosedIssuesCount = ghr.ClosedIssues()
		wg.Done()
	}()

	go func() {
		score.UpdatedIssuesCount = ghr.UpdatedIssues()
		score.CommentFrequency = ghr.CommentFrequency(score.UpdatedIssuesCount)
		wg.Done()
	}()

	go func() {
		score.DependentsCount = ghr.Dependents()
		wg.Done()
	}()

	wg.Wait()

	if ghr.Error != nil {
		return Score{}, ghr.Error
	}

	totalWeight := CreatedSinceWeight + UpdatedSinceWeight +
		ContributorCountWeight + OrgCountWeight +
		CommitFrequencyWeight + RecentReleasesWeight +
		ClosedIssuesWeight + UpdatedIssuesWeight +
		CommentFrequencyWeight + DependentsCountWeight +
		additionalParamsTotalWeight

	score.CriticalityScore = math.Round((ParamScore(score.CreatedSince, CreatedSinceThreshold, CreatedSinceWeight)+
		ParamScore(score.UpdatedSince, UpdatedSinceThreshold, UpdatedSinceWeight)+
		ParamScore(score.ContributorCount, ContributorCountThreshold, ContributorCountWeight)+
		ParamScore(score.OrgCount, OrgCountThreshold, OrgCountWeight)+
		ParamScore(score.CommitFrequency, CommitFrequencyThreshold, CommitFrequencyWeight)+
		ParamScore(score.RecentReleasesCount, RecentReleasesThreshold, RecentReleasesWeight)+
		ParamScore(score.ClosedIssuesCount, ClosedIssuesThreshold, ClosedIssuesWeight)+
		ParamScore(score.UpdatedIssuesCount, UpdatedIssuesThreshold, UpdatedIssuesWeight)+
		ParamScore(score.CommentFrequency, CommentFrequencyThreshold, CommentFrequencyWeight)+
		ParamScore(score.DependentsCount, DependentsCountThreshold, DependentsCountWeight)+
		additionalParamsScore)/totalWeight*100000) / 100000

	return score, nil
}

func PrintScore(score Score) {
	b, err := json.MarshalIndent(score, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
