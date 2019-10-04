package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"text/tabwriter"
	"time"
	"github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
)

var (
	token     string
	gitlabURL string
	groups    []string
	fromDate  time.Time
	untilDate time.Time
)

func main() {
	setConfig()

	logrus.Infof("Looking for MRs between %s and %s...", fromDate.Format("2006-01-02"), untilDate.Format("2006-01-02"))

	//Init Gitlab client
	git := gitlab.NewClient(nil, token)
	if err := git.SetBaseURL(gitlabURL); err != nil {
		logrus.Fatalf("Error when setting gitlab base URL : %s", err.Error())
	}

	print(findMergeRequest(git, groups, fromDate, untilDate))

}

//setUpLogs set on writer as output for the logs and set up the level
func setUpLogs(out io.Writer, level string) error {

	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	return nil
}

func setConfig() {
	viper.SetConfigName(".mergedlistr")
	viper.AddConfigPath("$HOME/")
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatalf("fatal error config file: %s", err)
	}

	token = viper.GetString("gitlab-token")
	gitlabURL = viper.GetString("gitlab-url")
	groups = viper.GetStringSlice("groups")

	//var duration time.Duration
	var fDate string
	var tDate string
	var verbosity string

	flag.StringVar(&fDate, "f", time.Now().AddDate(0, 0, -1).Format("2006-01-02") , `From date to look for merged PRs. Format is : "-f YYYY-MM-DD"`)
	flag.StringVar(&tDate, "t", time.Now().Format("2006-01-02") , `To date to look for merged PRs. Format is : "-t YYYY-MM-DD"`)
	flag.StringVar(&verbosity, "v", logrus.InfoLevel.String(), "Log level (debug, info, warn, error, fatal, panic")
	flag.Parse()

	fromDate, _ = time.Parse("2006-01-02", fDate)
	untilDate, _ = time.Parse("2006-01-02", tDate)
	untilDate = untilDate.AddDate(0, 0, 1)

	if err := setUpLogs(os.Stdout, verbosity); err != nil {
		logrus.Fatal(err.Error())
	}

}

func print(mergeRequestByProject map[string][]map[string]string) {
	//Init writer
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "Project\tMerge Request\tDate\tAuthor")
	for p, mrs := range mergeRequestByProject {
		for _, mr := range mrs {
			fmt.Fprintf(w, fmt.Sprintf("%s\t%s\t%s\t%s", p, mr["title"], mr["mergedAt"], mr["createdBy"]))
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintln(w)
	w.Flush()
}


func getGroups(git *gitlab.Client, groupToWatch []string) []*gitlab.Group {

	var wgg sync.WaitGroup
	mu := &sync.Mutex{}
	var groups []*gitlab.Group
	//Get groups to watch
	for _, gtg := range groupToWatch {
		wgg.Add(1)
		go func(gtg string) {
			gs, _, err := git.Groups.ListGroups(&gitlab.ListGroupsOptions{
				Search: gitlab.String(gtg),
			})
			if err != nil {
				logrus.Fatal(err)

			}
			for _, g := range gs {
				logrus.Debugf("Found group : %s", g.Name)
			}

			mu.Lock()
			groups = append(groups, gs...)
			mu.Unlock()
			wgg.Done()
		}(gtg)
	}
	wgg.Wait()

	return groups
}

func getProjects(git *gitlab.Client, groups []*gitlab.Group) []*gitlab.Project{
	var wgp sync.WaitGroup
	mu := &sync.Mutex{}
	var projects []*gitlab.Project

	for _, g := range groups {
		wgp.Add(1)
		go func(g *gitlab.Group) {
			archived := false
			//Find all projects
			ps, _, err := git.Groups.ListGroupProjects(g.ID, &gitlab.ListGroupProjectsOptions{
				//Since we want to avoid recurring across paginated results, we set the pagination very high
				ListOptions: gitlab.ListOptions{
					Page: 1,
					PerPage: 300,
				},
				Archived: &archived,
			})
			if err != nil {
				logrus.Fatal(err)
			}

			for _, p := range ps {
				logrus.Debugf("Found project %s in group %s", p.Name, g.Name)
			}

			mu.Lock()
			projects = append(projects, ps...)
			mu.Unlock()

			wgp.Done()
		}(g)
	}
	wgp.Wait()

	return projects
}

func getMergeRequests(git *gitlab.Client, projects []*gitlab.Project, from, to time.Time) []*gitlab.MergeRequest {

	var wgmr sync.WaitGroup
	mu := &sync.Mutex{}
	var mergeRequests []*gitlab.MergeRequest

	//We look for the from date minus 1 day because we gitlab does not offers th ability to filter on Merged At field.
	//But since merging is considered as an update, we prefer to take a little bit too large and filter on the merge date after.
	firstUpdatedDate := from.AddDate(0, 0, -1)
	lastUpdatedDate := to.AddDate(0, 0, 1)

	for _, p := range projects {
		wgmr.Add(1)
		go func(p *gitlab.Project) {
			defer wgmr.Done()
			logrus.Debugf("Looking for merge request in project : %s", p.Name)
			mrs, _, err := git.MergeRequests.ListProjectMergeRequests(p.ID, &gitlab.ListProjectMergeRequestsOptions{
				ListOptions: gitlab.ListOptions{
					Page: 1,
					PerPage: 300,
				},
				State:       gitlab.String("merged"),
				OrderBy:     gitlab.String("updated_at"),
				Scope:       gitlab.String("all"),
				UpdatedAfter: &firstUpdatedDate,
				UpdatedBefore: &lastUpdatedDate,

			})

			if err != nil {
				logrus.Fatal(err)
			}

			logrus.Debugf("Found %d MR in project %s", len(mrs), p.Name)

			for _, mr := range mrs {
				logrus.Debugf("Found Merge Request %d titled %s on project %s", mr.ID, mr.Title, p.Name)
			}

			mu.Lock()
			mergeRequests = append(mergeRequests, mrs...)
			mu.Unlock()


		}(p)
	}
	wgmr.Wait()
	return mergeRequests
}


func getProjectName(ID int, projects []*gitlab.Project) string {

	for _, p := range projects {
		if p.ID == ID {
			return p.Name
		}
	}

	return "Unknown Project"
}

func findMergeRequest(git *gitlab.Client, groupToWatch []string, from, to time.Time) map[string][]map[string]string {
	results := make(map[string][]map[string]string, 500)

	groups := getGroups(git, groupToWatch)
	logrus.Debugf("Found %d groups", len(groups))
	projects := getProjects(git, groups)
	logrus.Debugf("Found %d projects", len(projects))
	mergeRequests := getMergeRequests(git, projects, from, to)

	for _, mr := range mergeRequests {

		if mr.MergedAt.After(from) && mr.MergedAt.Before(to) {
			mr := map[string]string{
				"projectName": getProjectName(mr.ProjectID, projects),
				"title":       mr.Title,
				"mergedAt":    mr.MergedAt.Format(time.RFC3339),
				"createdBy":   mr.Author.Name,
				"link": mr.WebURL,
			}
			results[mr["projectName"]] = append(results[mr["projectName"]], mr)
		}
	}

	return results
}
