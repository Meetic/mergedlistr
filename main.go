package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
)

var (
	token     string
	gitlabURL string
	groups    []string
	fromDate  time.Time
)

func main() {
	setConfig()

	fmt.Println("Searching for PRs...")

	//Init Gitlab client
	git := gitlab.NewClient(nil, token)
	git.SetBaseURL(gitlabURL)

	print(getMergeRequests(git, groups, fromDate))

}

func setConfig() {
	viper.SetConfigName(".mergedlistr")
	viper.AddConfigPath("$HOME/")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf(("fatal error config file: %s", err)
	}

	token = viper.GetString("gitlab-token")
	gitlabURL = viper.GetString("gitlab-url")
	groups = viper.GetStringSlice("groups")

	var duration time.Duration
	flag.DurationVar(&duration, "t", 24*time.Hour, `Duration to look for merges PRs. Example :  "-t 24h", "-t 15m", or "-t 30s"`)
	flag.Parse()

	fromDate = time.Now().Add(-duration)

}

func print(mergeRequestByProject map[string][]map[string]string) {
	//Init writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "Project\tMerge Request\tDate\tAuthor\t")
	for p, mrs := range mergeRequestByProject {
		for _, mr := range mrs {
			fmt.Fprintf(w, fmt.Sprintf("%s\t%s\t%s\t%s\t", p, mr["title"], mr["mergedAt"], mr["createdBy"]))
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintln(w)
	w.Flush()
}

func getMergeRequests(git *gitlab.Client, groupToWatch []string, from time.Time) map[string][]map[string]string {

	groupsCh := make(chan *gitlab.Group, len(groupToWatch))
	projectsList := map[int]*gitlab.Project{}
	mu := &sync.Mutex{}

	results := make(map[string][]map[string]string, 500)

	var wgg sync.WaitGroup
	//Get groups to watch
	for _, gtg := range groupToWatch {
		wgg.Add(1)
		go func(gtg string) {
			gs, _, err := git.Groups.ListGroups(&gitlab.ListGroupsOptions{
				Search: gitlab.String(gtg),
			})
			if err != nil {
				log.Fatal(err)
			}

			for _, g := range gs {
				groupsCh <- g
			}
			wgg.Done()
		}(gtg)
	}
	wgg.Wait()
	close(groupsCh)

	var wgp sync.WaitGroup
	for g := range groupsCh {
		wgp.Add(1)
		go func(g *gitlab.Group) {
			//Find all projects
			projects, _, err := git.Groups.ListGroupProjects(g.ID, &gitlab.ListGroupProjectsOptions{
				gitlab.ListOptions{
					PerPage: 100,
				},
			})
			if err != nil {
				log.Fatal(err)
			}

			for _, p := range projects {
				mu.Lock()
				projectsList[p.ID] = p
				mu.Unlock()
			}
			wgp.Done()
		}(g)
	}
	wgp.Wait()

	pageOptions := gitlab.ListOptions{
		PerPage: 100,
	}

	mrOptions := &gitlab.ListMergeRequestsOptions{
		ListOptions: pageOptions,
		State:       gitlab.String("merged"),
		OrderBy:     gitlab.String("updated_at"),
		Scope:       gitlab.String("all"),
	}

	mergeRequests, _, err := git.MergeRequests.ListMergeRequests(mrOptions)

	if err != nil {
		log.Fatal(err)
	}

	for _, mr := range mergeRequests {
		if p, ok := projectsList[mr.ProjectID]; ok {
			if mr.UpdatedAt.After(from) {
				mr := map[string]string{
					"projectName": p.Name,
					"title":       mr.Title,
					"mergedAt":    mr.UpdatedAt.Format(time.RFC3339),
					"createdBy":   mr.Author.Name,
				}

				results[mr["projectName"]] = append(results[mr["projectName"]], mr)
			}
		}
	}

	return results
}
