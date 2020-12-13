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

import "regexp"

// Constants used in OSS criticality score calculation.

const (

	// Weights for various parameters.

	CreatedSinceWeight     = 1.0
	UpdatedSinceWeight     = -1.0
	ContributorCountWeight = 2.0
	OrgCountWeight         = 1.0
	CommitFrequencyWeight  = 1.0
	RecentReleasesWeight   = 0.5
	ClosedIssuesWeight     = 0.5
	UpdatedIssuesWeight    = 0.5
	CommentFrequencyWeight = 1.0
	DependentsCountWeight  = 2.0

	// Max thresholds for various parameters.

	CreatedSinceThreshold     = 120.0
	UpdatedSinceThreshold     = 120.0
	ContributorCountThreshold = 5000.0
	OrgCountThreshold         = 10.0
	CommitFrequencyThreshold  = 1000.0
	RecentReleasesThreshold   = 26.0
	ClosedIssuesThreshold     = 5000.0
	UpdatedIssuesThreshold    = 5000.0
	CommentFrequencyThreshold = 15.0
	DependentsCountThreshold  = 500000.0

	// Others.

	TopContributorCount = 15
	IssueLookbackDays   = 90
	ReleaseLookbackDays = 365
)

var DependentsRegex *regexp.Regexp

func init() {
	// Regex to match dependents count.
	DependentsRegex = regexp.MustCompile(".*[^0-9,]([0-9,]+).*commit results")
}
