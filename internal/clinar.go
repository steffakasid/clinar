package internal

import (
	"regexp"
	"strconv"
	"sync"

	logger "github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

const runnerState = "offline"

type Clinar struct {
	*gitlab.Client
	StaleRunnerIDs []*gitlab.RunnerDetails
	ExcludeFilter  []string
	IncludePattern *regexp.Regexp
}

func (c *Clinar) appendRunnerIds(rners []*gitlab.Runner) {
	for _, rner := range rners {
		details, _, err := c.Runners.GetRunnerDetails(rner.ID)
		if err != nil {
			logger.Errorf("Error %s getting runner details for runner ID %d", err, rner.ID)
		}
		grpsNprojs := []abstractRunnerLocation{}
		for _, grp := range details.Groups {
			grpsNprojs = append(grpsNprojs, abstractRunnerLocation{grp.ID, grp.Name})
		}
		for _, proj := range details.Projects {
			grpsNprojs = append(grpsNprojs, abstractRunnerLocation{proj.ID, proj.Name})
		}
		if c.isFilteredOut(grpsNprojs) {
			logger.Infof("Skipping %d", rner.ID)
		} else {
			if c.isIncluded(grpsNprojs) {
				c.StaleRunnerIDs = append(c.StaleRunnerIDs, details)
			}
		}
	}
}

func (c *Clinar) GetAllRunners() error {
	opts := &gitlab.ListRunnersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Status: gitlab.String(runnerState),
	}

	rners, resp, err := c.Runners.ListRunners(opts)
	if err != nil {
		return err
	}
	c.appendRunnerIds(rners)

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
			logger.Error(err)
		}
		c.appendRunnerIds(rners)
	}

	return nil
}

type listRunnerResultWrapper struct {
	rners []*gitlab.Runner
	err   error
}

func (c Clinar) wrapListRunners(opts gitlab.ListRunnersOptions, results chan<- listRunnerResultWrapper, wg *sync.WaitGroup) {
	rners, _, err := c.Runners.ListRunners(&opts)
	results <- listRunnerResultWrapper{rners, err}
	wg.Done()
}

func (c *Clinar) CleanupRunners() error {
	if len(c.StaleRunnerIDs) <= 0 {
		err := c.GetAllRunners()
		if err != nil {
			return err
		}
	}

	if len(c.StaleRunnerIDs) == 0 {
		logger.Info("No runners to be purged!")
		return nil
	}

	result := make(chan responseWrapper, len(c.StaleRunnerIDs))
	var wg sync.WaitGroup
	for _, rner := range c.StaleRunnerIDs {
		wg.Add(1)
		c.wrapDeleteRegisteredRunnerById(*rner, result, &wg)
	}
	wg.Wait()
	close(result)

	for deleteResult := range result {
		if deleteResult.err != nil {
			logger.Error(deleteResult.err)
		}
		logger.Debugf("DeleteRegisteredRunnerByID returned status %s\n", deleteResult.resp.Status)
	}

	return nil
}

type responseWrapper struct {
	resp gitlab.Response
	err  error
}

func (c Clinar) wrapDeleteRegisteredRunnerById(rner gitlab.RunnerDetails, result chan<- responseWrapper, wg *sync.WaitGroup) {
	logger.Infof("Deleting %d - %s", rner.ID, rner.Name)
	resp, err := c.Runners.DeleteRegisteredRunnerByID(rner.ID)
	result <- responseWrapper{*resp, err}
	wg.Done()
}

type abstractRunnerLocation struct {
	id   int
	name string
}

func (c Clinar) isFilteredOut(locations []abstractRunnerLocation) bool {
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
