package internal

import (
	"regexp"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const runnerState = "offline"

type GitLabClient interface {
	GetRunnerDetails(rid interface{}, options ...gitlab.RequestOptionFunc) (*gitlab.RunnerDetails, *gitlab.Response, error)
	ListRunners(opt *gitlab.ListRunnersOptions, options ...gitlab.RequestOptionFunc) ([]*gitlab.Runner, *gitlab.Response, error)
	DeleteRegisteredRunnerByID(rid int, options ...gitlab.RequestOptionFunc) (*gitlab.Response, error)
}

type Clinar struct {
	Client         GitLabClient
	Logger         *logrus.Logger
	ExcludeFilter  []string       `mapstructure:"exclude"`
	IncludePattern *regexp.Regexp `mapstructure:"include"`
}

// GetRunnerDetails return the gitlab.RunnerDetails for all given []*gitlab.Runner
func (c *Clinar) GetRunnerDetails(rners []*gitlab.Runner) []*gitlab.RunnerDetails {
	runnerDetails := []*gitlab.RunnerDetails{}
	// TODO: We could get Details in Chunks with goroutines
	for _, rner := range rners {
		details, _, err := c.Client.GetRunnerDetails(rner.ID)
		if err != nil {
			c.Logger.Errorf("Error %s getting runner details for runner ID %d", err, rner.ID)
		} else {
			grpsNprojs := []abstractRunnerLocation{}
			for _, grp := range details.Groups {
				grpsNprojs = append(grpsNprojs, abstractRunnerLocation{grp.ID, grp.Name})
			}
			for _, proj := range details.Projects {
				grpsNprojs = append(grpsNprojs, abstractRunnerLocation{proj.ID, proj.Name})
			}
			if c.isExcluded(grpsNprojs) {
				c.Logger.Infof("Skipping %d", rner.ID)
			} else {
				if c.isIncluded(grpsNprojs) {
					runnerDetails = append(runnerDetails, details)
				}
			}
		}
	}
	return runnerDetails
}

func (c *Clinar) GetAllRunners() ([]*gitlab.Runner, error) {
	runners := []*gitlab.Runner{}

	opts := &gitlab.ListRunnersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Status: gitlab.Ptr(runnerState),
	}

	rners, resp, err := c.Client.ListRunners(opts)
	if err != nil {
		return nil, err
	}
	runners = append(runners, rners...)

	results := make(chan listRunnerResultWrapper, resp.TotalPages)
	var wg sync.WaitGroup
	for i := 2; i <= resp.TotalPages; i++ {
		opts.Page = i
		wg.Add(1)
		go c.wrapListRunners(*opts, results, &wg)
	}
	wg.Wait()
	close(results)

	for rnerResult := range results {
		if rnerResult.err != nil {
			c.Logger.Error(rnerResult.err)
		} else {
			runners = append(runners, rnerResult.rners...)
		}
	}

	return runners, nil
}

func (c Clinar) wrapListRunners(opts gitlab.ListRunnersOptions, results chan<- listRunnerResultWrapper, wg *sync.WaitGroup) {
	rners, _, err := c.Client.ListRunners(&opts)
	results <- listRunnerResultWrapper{rners, err}
	wg.Done()
}

func (c *Clinar) CleanupRunners(staleRunnerIDs []*gitlab.RunnerDetails) {
	if len(staleRunnerIDs) == 0 {
		c.Logger.Info("No runners to be purged!")
	}

	result := make(chan responseWrapper, len(staleRunnerIDs))
	var wg sync.WaitGroup
	for _, rner := range staleRunnerIDs {
		c.Logger.Infof("Deleting %d - %s", rner.ID, rner.Name)
		wg.Add(1)
		c.wrapDeleteRegisteredRunnerById(*rner, result, &wg)
	}
	wg.Wait()
	close(result)

	for deleteResult := range result {
		if deleteResult.err != nil {
			c.Logger.Error(deleteResult.err)
		}
		c.Logger.Debugf("DeleteRegisteredRunnerByID returned status %s\n", deleteResult.resp.Status)
	}
}

func (c Clinar) wrapDeleteRegisteredRunnerById(rner gitlab.RunnerDetails, result chan<- responseWrapper, wg *sync.WaitGroup) {
	resp, err := c.Client.DeleteRegisteredRunnerByID(rner.ID)
	result <- responseWrapper{*resp, err}
	wg.Done()
}

func (c Clinar) isExcluded(locations []abstractRunnerLocation) bool {
	for _, filter := range c.ExcludeFilter {
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

func (c Clinar) isIncluded(locations []abstractRunnerLocation) bool {
	if c.IncludePattern == nil {
		return true
	} else {
		for _, loc := range locations {
			if c.IncludePattern.MatchString(loc.name) {
				return true
			}
		}
	}
	return false
}
