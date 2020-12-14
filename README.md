# criticalityscore

This is a Go port of the official Python-based [ossf/criticality_score](https://github.com/ossf/criticality_score) tool.

## Command-line Usage

First, install the command-line tool.
```bash
go get -u github.com/engelsjk/criticalityscore
```

Next, run the criticalityscore tool with a specified GitHub repository URL.

```bash
criticalityscore --repo https://github.com/kubernetes/kubernetes
```

Output:
```bash
name: kubernetes
url: https://github.com/kubernetes/kubernetes
language: Go
created_since: 79
updated_since: 0
contributor_count: 3674
org_count: 5
commit_frequency: 101.4
recent_releases_count: 76
closed_issues_count: 2902
updated_issues_count: 5129
comment_frequency: 5.7
dependents_count: 407344
criticality_score: 0.98606
```
