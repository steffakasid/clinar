package internal

import (
	"fmt"
	"strconv"

	"github.com/xanzy/go-gitlab"
)

type Clinar struct {
	*gitlab.Client
	StaleRunnerIDs []*gitlab.RunnerDetails
	Filter         []string
}

func (r *Clinar) appendRunnerIds(rners []*gitlab.Runner) {
	for _, rner := range rners {
		details, _, err := r.Runners.GetRunnerDetails(rner.ID)
		if err != nil {
			fmt.Printf("Error %s getting runner details for runner ID %d\n", err, rner.ID)
		}
		r.StaleRunnerIDs = append(r.StaleRunnerIDs, details)
	}
}

func (c *Clinar) GetAllRunners() error {
	opts := &gitlab.ListRunnersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Status: gitlab.String("offline"),
	}

	for {
		rners, resp, err := c.Runners.ListRunners(opts)
		if err != nil {
			return err
		}

		c.appendRunnerIds(rners)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opts.Page = resp.NextPage
	}
	return nil
}

func (c *Clinar) CleanupRunners() error {
	if len(c.StaleRunnerIDs) <= 0 {
		err := c.GetAllRunners()
		if err != nil {
			return err
		}
	}

	if len(c.StaleRunnerIDs) == 0 {
		fmt.Println("No runners to be purged!")
		return nil
	}

	for _, rner := range c.StaleRunnerIDs {
		grpsNprojs := []abstractRunnerLocation{}
		for _, grp := range rner.Groups {
			grpsNprojs = append(grpsNprojs, abstractRunnerLocation{grp.ID, grp.Name})
		}
		for _, proj := range rner.Projects {
			grpsNprojs = append(grpsNprojs, abstractRunnerLocation{proj.ID, proj.Name})
		}
		if c.isFilteredOut(grpsNprojs) {
			fmt.Printf("Skipping %d", rner.ID)
		} else {
			fmt.Printf("Deleting %d - %s", rner.ID, rner.Name)
			resp, err := c.Runners.DeleteRegisteredRunnerByID(rner.ID)
			if err != nil {
				return err
			}
			fmt.Printf("returned status %s\n", resp.Status)
		}
	}
	return nil
}

type abstractRunnerLocation struct {
	id   int
	name string
}

func (c Clinar) isFilteredOut(locations []abstractRunnerLocation) bool {
	for _, filter := range c.Filter {
		for _, loc := range locations {
			if filter == loc.name {
				return true
			} else if filter == strconv.Itoa(loc.id) {
				return true
			}
		}
	}
	return false
}
