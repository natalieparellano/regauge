package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v30/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type PullRequest struct {
	Repo                 string
	CreatedAt            time.Time
	URL                  string
	Title                string
	Username             string
	State                string
	FirstReviewCommentAt time.Time
	TimeToFirstReview    time.Duration
}

func (r PullRequest) Headers() []string {
	return []string{"Repo", "CreatedAt", "URL", "Title", "Username", "State", "FirstReviewCommentAt", "TimeToFirstReview"}
}

func (r PullRequest) Values() []string {
	return []string{r.Repo, r.CreatedAt.String(), r.URL, r.Title, r.Username, r.State, r.FirstReviewCommentAt.String(), r.TimeToFirstReview.String()}
}

func main() {
	repoList := os.Args[1]
	log.Printf("processing repos: %s\n", repoList)
	outputFile := os.Args[2]
	log.Printf("output file: %s\n", outputFile)

	githubClient := newGithubClient()

	var pullRequests []PullRequest
	for _, repo := range strings.Split(repoList, ",") {
		result, err := queryPRs(githubClient, repo)
		if err != nil {
			log.Fatal(err)
		}
		pullRequests = append(pullRequests, result...)
	}

	if err := writeCsv(pullRequests, outputFile); err != nil {
		log.Fatal(err)
	}
}

func newGithubClient() *github.Client {
	token := os.Getenv("GITHUB_TOKEN")
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: token,
	})
	httpClient := oauth2.NewClient(context.TODO(), tokenSource)
	return github.NewClient(httpClient)
}

func queryPRs(githubClient *github.Client, repo string) ([]PullRequest, error) {
	parts := strings.Split(repo, "/")
	owner := parts[0]
	repoName := parts[1]

	prList, _, err := githubClient.PullRequests.List(context.TODO(), owner, repoName, &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			//Page: 2 // Uncomment to get older data
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("listing pull requests for %s", repo))
	}

	var pullRequests []PullRequest
	for _, githubPR := range prList {
		pr := PullRequest{
			Repo:      repo,
			CreatedAt: *githubPR.CreatedAt,
			URL:       *githubPR.URL,
			Title:     *githubPR.Title,
			Username:  *githubPR.User.Login,
			State:     *githubPR.State,
		}

		reviews, _, err := githubClient.PullRequests.ListReviews(context.TODO(), owner, repoName, *githubPR.Number, &github.ListOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "listing review comments")
		}
		for _, review := range reviews {
			if *review.User.Login == *githubPR.User.Login {
				continue
			}
			pr.FirstReviewCommentAt = *review.SubmittedAt
			pr.TimeToFirstReview = review.SubmittedAt.Sub(*githubPR.CreatedAt)
			break
		}

		pullRequests = append(pullRequests, pr)
	}

	return pullRequests, nil
}

func writeCsv(pullRequests []PullRequest, filepath string) error {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "opening file")
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	for idx, pr := range pullRequests {
		if idx == 0 {
			if err := writer.Write(pr.Headers()); err != nil {
				return errors.Wrap(err, "writing headers")
			}
		}
		if err := writer.Write(pr.Values()); err != nil {
			return errors.Wrap(err, "writing values")
		}
	}
	return nil
}
