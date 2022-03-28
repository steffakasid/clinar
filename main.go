package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/briandowns/spinner"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/steffakasid/clinar/internal"
	"github.com/xanzy/go-gitlab"
)

var clinar *internal.Clinar = &internal.Clinar{}

func init() {
	flag.BoolP(APPROVE, "a", false, "Acknowledge to purge all stale runners")
	flag.StringArrayP(EXCLUDE, "e", nil, "Filter out runners with specified groups/projects. Filter can be given by id or name. Exclude takes precedences before include.")
	flag.StringP(INCLUDE, "i", "", "Regular expression include filter. Matches on project and group names. If runner is set one group or project this runner will be included.")

	flag.Usage = func() {
		w := os.Stderr

		fmt.Fprintf(w, "Usage of %s: \n", os.Args[0])
		fmt.Fprintln(w, `
This tool basically get's all offline runners which a user can administer. 
If you don't provide the '--approve' flag the tool just shows all runners 
which are offline with some additional information. After you provide the
'--approve' flag all offline runners are deleted.

Usage:
  clinar [flags]

Variables:
  - GITLAB_TOKEN   - the GitLab token to access the Gitlab instance
  - GITLAB_HOST    - the GitLab host which should be accessed [Default: https://gitlab.com]

Examples:
  clinar                       - get all stale runners which can be administred by the GITLAB_TOKEN
  clinar --approve             - cleanup all stal runners which can be administred by the GITLAB_TOKEN 
  clinar --exclude 1234        - get all stale runners which can be administred by the GITLAB_TOKEN. Excluding project or group with ID 1234.
  clinar --include ^prefix.*   - get alle stale runners which are set on a group / project where the name matches ^prefix.*

Flags:`)

		flag.PrintDefaults()
	}

	flag.Parse()
	err := viper.BindPFlags(flag.CommandLine)
	if err != nil {
		logger.Fatal(err)
	}
	InitConfig()
}

func main() {
	var err error

	if viper.GetString(GTILAB_TOKEN) == "" {
		log.Fatal("GITLAB_TOKEN env var not set")
	} else {
		gitLabClient, err := gitlab.NewClient(viper.GetString(GTILAB_TOKEN), gitlab.WithBaseURL(viper.GetString(GITLAB_HOST)))
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		clinar.Client = gitLabClient.Runners
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Start()
	err = clinar.GetAllRunners()
	if err != nil {
		panic(err)
	}
	if viper.GetBool(APPROVE) {
		if err != nil {
			panic(err)
		}
	} else {
		printFoundRunners()
	}
	s.Stop()
}

func printFoundRunners() {
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
