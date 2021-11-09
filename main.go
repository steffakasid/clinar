package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/briandowns/spinner"
	flag "github.com/spf13/pflag"
	"github.com/steffakasid/clinar/internal"
	"github.com/xanzy/go-gitlab"
)

var (
	approve        bool
	excludeFilter  []string
	includePattern string
)

var clinar *internal.Clinar = &internal.Clinar{
	StaleRunnerIDs: []*gitlab.RunnerDetails{},
}

func init() {
	flag.BoolVarP(&approve, "approve", "a", false, "Acknowledge to purge all stale runners")
	flag.StringArrayVarP(&excludeFilter, "exclude", "e", nil, "Filter out runners with specified groups/projects. Filter can be given by id or name. Exclude takes precedences before include.")
	flag.StringVarP(&includePattern, "include", "i", "", "Regular expression include filter. Matches on project and group names. If runner is set one group or project this runner will be included.")
	flag.Parse()
}

func main() {
	var err error

	gitHost := os.Getenv("GITLAB_HOST")
	if gitHost == "" {
		gitHost = "https://gitlab.com"
	}

	if token, exists := os.LookupEnv("GITLAB_TOKEN"); !exists {
		log.Fatal("GITLAB_TOKEN env var not set")
	} else {
		clinar.Client, err = gitlab.NewClient(token, gitlab.WithBaseURL(gitHost))
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
	}

	if len(excludeFilter) > 0 {
		clinar.ExcludeFilter = excludeFilter
	}

	if includePattern != "" {
		rex, err := regexp.Compile(includePattern)
		if err != nil {
			panic(err)
		}
		clinar.IncludePattern = rex
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Start()
	err = clinar.GetAllRunners()
	if err != nil {
		panic(err)
	}
	if approve {
		err = clinar.CleanupRunners()
		if err != nil {
			panic(err)
		}
	} else {
		printFoundRunenrs()
	}
	s.Stop()
}

func printFoundRunenrs() {
	if len(clinar.StaleRunnerIDs) > 0 {
		fmt.Println()
		for _, rner := range clinar.StaleRunnerIDs {
			groups := []string{}
			for _, grp := range rner.Groups {
				groups = append(groups, grp.Name)
			}
			projects := []string{}
			for _, proj := range rner.Projects {
				projects = append(projects, proj.Name)
			}
			fmt.Printf("%d - %s - %s - %t - %s - %s\n", rner.ID, rner.RunnerType, rner.Description, rner.Online, groups, projects)
		}
	} else {
		fmt.Println("No stale runners found!")
	}
}
