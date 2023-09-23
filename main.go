package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v55/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a report given a start date and end date in RFC 3339 format",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		startDate, _ := time.Parse(time.DateOnly, args[0])
		endDate, _ := time.Parse(time.DateOnly, args[1])
		var users []string
		ctx := context.Background()
		// create http client with personal access token
		httpClient := &http.Client{}

		client := github.NewClient(httpClient).WithAuthToken(viper.GetString("github.token"))

		if err := viper.UnmarshalKey("github.users", &users); err != nil {
			panic(err)
		}
		for _, u := range users {
			gUsr, err := GetUserDetails(ctx, client, u)
			if err != nil {
				fmt.Printf("error geting user details: %v\n", err)
				continue
			}
			ud := UserDetails{
				UserName: u,
				Name:     *gUsr.Name,
			}
			events, err := GetEventDetails(ctx, client, u)
			if err != nil {
				fmt.Printf("error getting event details: %v\n", err)
				continue
			}
			// Count the number of unique repositories.
			repoSet := make(map[string]struct{})
			for _, event := range events {
				if event.CreatedAt.After(startDate) && event.CreatedAt.Before(endDate) {
					if *event.Type == "PullRequestReviewEvent" {
						ud.NumPullRequestReviews++
					}
					if *event.Type == "PullRequestEvent" {
						ud.NumPullRequests++
					}
					if *event.Type == "PushEvent" {
						ud.NumCommits++
					}
					repoSet[*event.Repo.Name] = struct{}{}
				}
			}
			ud.NumRepos = len(repoSet)
			fmt.Printf("ud: %+v\n", ud)
		}
		// Write logic here to pull user details from Github v3 API over the given date range and generate the report

		fmt.Printf("Report generated from %s to %s\n", startDate.Format(time.DateOnly), endDate.Format(time.DateOnly))
	},
}

type UserDetails struct {
	UserName              string
	Name                  string
	NumCommits            int
	NumPullRequests       int
	NumPullRequestReviews int
	NumRepos              int
}

func GetUserDetails(ctx context.Context, client *github.Client, username string) (*github.User, error) {
	usr, resp, err := client.Users.Get(ctx, username)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("unexepected status code got %d", resp.StatusCode)
	}

	return usr, nil
}

func GetEventDetails(ctx context.Context, client *github.Client, username string) ([]*github.Event, error) {
	events, resp, err := client.Activity.ListEventsPerformedByUser(ctx, username, false, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("unexepected status code got %d", resp.StatusCode)
	}

	return events, nil
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".config")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w ", err))
	}

	rootCmd := &cobra.Command{Use: "bigbrother"}

	rootCmd.AddCommand(reportCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
