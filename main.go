package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/briandowns/spinner"
	flag "github.com/spf13/pflag"
	"github.com/steffakasid/clinar/internal"
	"github.com/xanzy/go-gitlab"
)

var (
	approve bool
)

var clinar *internal.Clinar = &internal.Clinar{
	GroupIDs:       []int{},
	ProjectIDs:     []int{},
	StaleRunnerIDs: []*gitlab.RunnerDetails{},
}

func init() {
	flag.BoolVarP(&approve, "approve", "a", false, "Acknowledge to purge all stale runners")
	flag.Parse()
}

func main() {
	var err error
	if token, exists := os.LookupEnv("GITLAB_TOKEN"); !exists {
		log.Fatal("GITLAB_TOKEN env var not set")
	} else {
		clinar.Client, err = gitlab.NewClient(token)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
	}

	// err = clinar.GetAllGroups()
	// if err != nil {
	// 	panic(err)
	// }

	// err = clinar.GetAllProjects()
	// if err != nil {
	// 	panic(err)
	// }
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
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
		if len(clinar.StaleRunnerIDs) > 0 {
			for _, rner := range clinar.StaleRunnerIDs {
				fmt.Printf("%d - %s - %s - %t\n", rner.ID, rner.RunnerType, rner.Description, rner.Online)
			}
		} else {
			fmt.Println("No stale runners found!")
		}
	}
	s.Stop()
}
