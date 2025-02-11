package internal

import gitlab "gitlab.com/gitlab-org/api/client-go"

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
