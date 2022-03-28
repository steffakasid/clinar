package internal

import "github.com/xanzy/go-gitlab"

type responseWrapper struct {
	resp gitlab.Response
	err  error
}

type listRunnerResultWrapper struct {
	rners []*gitlab.Runner
	err   error
}

type abstractRunnerLocation struct {
	id   int
	name string
}
