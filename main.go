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

var excludeFilter []string

var clinar *internal.Clinar = &internal.Clinar{
	StaleRunnerIDs: []*gitlab.RunnerDetails{},
}

func init() {
	flag.BoolP(internal.APPROVE, "a", false, "Acknowledge to purge all stale runners")
	flag.StringArrayVarP(&excludeFilter, "exclude", "e", nil, "Filter out runners with specified groups/projects. Filter can be given by id or name. Exclude takes precedences before include.")
	flag.StringP(internal.INCLUDE, "i", "", "Regular expression include filter. Matches on project and group names. If runner is set one group or project this runner will be included.")

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
	viper.BindPFlags(flag.CommandLine)
	internal.InitConfig()
}

func main() {
	var err error

	if viper.GetString(internal.GTILAB_TOKEN) == "" {
		log.Fatal("GITLAB_TOKEN env var not set")
	} else {
		clinar.Client, err = gitlab.NewClient(viper.GetString(internal.GTILAB_TOKEN), gitlab.WithBaseURL(viper.GetString(internal.GITLAB_HOST)))
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
	}

	if len(excludeFilter) > 0 {
		clinar.ExcludeFilter = excludeFilter
	}

	if viper.GetString(internal.INCLUDE) != "" {
		rex, err := regexp.Compile(viper.GetString(internal.INCLUDE))
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
	if viper.GetBool(internal.APPROVE) {
		err = clinar.CleanupRunners()
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
