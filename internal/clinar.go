package internal

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

type Clinar struct {
	*gitlab.Client
	GroupIDs       []int
	ProjectIDs     []int
	StaleRunnerIDs []*gitlab.RunnerDetails
}

func (r *Clinar) appendGrpIds(grps []*gitlab.Group) {
	for _, grp := range grps {
		r.GroupIDs = append(r.GroupIDs, grp.ID)
	}
}

func (r *Clinar) appendProjIds(projs []*gitlab.Project) {
	for _, proj := range projs {
		r.ProjectIDs = append(r.ProjectIDs, proj.ID)
	}
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

func (c *Clinar) GetAllGroups() error {
	opts := &gitlab.ListGroupsOptions{
		Owned: gitlab.Bool(true),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		grps, resp, err := c.Groups.ListGroups(opts)
		if err != nil {
			return err
		}
		c.appendGrpIds(grps)
		// Exit the loop when we've seen all pages.
		if resp.CurrentPage >= resp.TotalPages {
			break
		}
		// Update the page number to get the next page.
		opts.Page = resp.NextPage
	}
	return nil
}

func (c *Clinar) GetAllProjects() error {
	opts := &gitlab.ListProjectsOptions{
		Archived: gitlab.Bool(false),
		Owned:    gitlab.Bool(true),
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		projs, resp, err := c.Projects.ListProjects(opts)
		if err != nil {
			return err
		}
		c.appendProjIds(projs)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opts.Page = resp.NextPage
	}
	return nil
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
		fmt.Printf("Deleting %d ", rner.ID)
		resp, err := c.Runners.DeleteRegisteredRunnerByID(rner.ID)
		if err != nil {
			return err
		}
		fmt.Printf("returned status %s\n", resp.Status)
	}
	return nil
}
