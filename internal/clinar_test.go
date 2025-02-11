package internal

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/sirupsen/logrus"
	logrusTest "github.com/sirupsen/logrus/hooks/test"
	"github.com/steffakasid/clinar/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

func TestGetRunnerDetails(t *testing.T) {
	logger, logHook := logrusTest.NewNullLogger()

	t.Run("Simple case", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		mockGetRunnerDetails(mock, 1)
		clinar := Clinar{Client: mock, Logger: logger}
		details := clinar.GetRunnerDetails([]*gitlab.Runner{{ID: 1}})
		assert.Len(t, details, 1)
		assert.Equal(t, "someRunner1", details[0].Name)
		mock.AssertExpectations(t)
	})

	t.Run("Filter out project by name", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		mockGetRunnerDetails(mock, 3)
		clinar := Clinar{Client: mock, Logger: logger, ExcludeFilter: []string{"Project2"}}
		details := clinar.GetRunnerDetails([]*gitlab.Runner{{ID: 1}, {ID: 2}, {ID: 3}})
		assert.Len(t, details, 2)
		assert.Equal(t, "someRunner1", details[0].Name)
		assert.Equal(t, "someRunner3", details[1].Name)
		mock.AssertExpectations(t)
	})

	t.Run("Filter out project by id", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		mockGetRunnerDetails(mock, 3)
		clinar := Clinar{Client: mock, Logger: logger, ExcludeFilter: []string{"22"}}
		details := clinar.GetRunnerDetails([]*gitlab.Runner{{ID: 1}, {ID: 2}, {ID: 3}})
		assert.Len(t, details, 2)
		assert.Equal(t, "someRunner1", details[0].Name)
		assert.Equal(t, "someRunner3", details[1].Name)
		mock.AssertExpectations(t)
	})

	t.Run("Filter out group by name", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		mockGetRunnerDetails(mock, 3)
		clinar := Clinar{Client: mock, Logger: logger, ExcludeFilter: []string{"Group1"}}
		details := clinar.GetRunnerDetails([]*gitlab.Runner{{ID: 1}, {ID: 2}, {ID: 3}})
		assert.Len(t, details, 2)
		assert.Equal(t, "someRunner2", details[0].Name)
		assert.Equal(t, "someRunner3", details[1].Name)
		mock.AssertExpectations(t)
	})

	t.Run("Filter out group by id", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		mockGetRunnerDetails(mock, 3)
		clinar := Clinar{Client: mock, Logger: logger, ExcludeFilter: []string{"11"}}
		details := clinar.GetRunnerDetails([]*gitlab.Runner{{ID: 1}, {ID: 2}, {ID: 3}})
		assert.Len(t, details, 2)
		assert.Equal(t, "someRunner2", details[0].Name)
		assert.Equal(t, "someRunner3", details[1].Name)
		mock.AssertExpectations(t)
	})

	t.Run("Include filter", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		mockGetRunnerDetails(mock, 4)
		clinar := Clinar{Client: mock, Logger: logger, IncludePattern: regexp.MustCompile(".*roject[3,4]")}
		details := clinar.GetRunnerDetails([]*gitlab.Runner{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}})
		assert.Len(t, details, 2)
		assert.Equal(t, "someRunner3", details[0].Name)
		assert.Equal(t, "someRunner4", details[1].Name)
		mock.AssertExpectations(t)
	})

	t.Run("Error from GetRunnerDetails", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		logHook.Reset()
		mock.EXPECT().GetRunnerDetails(1).Return(nil, &gitlab.Response{}, errors.New("Something went wrong")).Once()
		clinar := Clinar{Client: mock, Logger: logger}
		details := clinar.GetRunnerDetails([]*gitlab.Runner{{ID: 1}})
		assert.Len(t, details, 0)
		assert.Len(t, logHook.Entries, 1)
		assert.Equal(t, "Error Something went wrong getting runner details for runner ID 1", logHook.Entries[0].Message)
		assert.Equal(t, logrus.ErrorLevel, logHook.Entries[0].Level)
		mock.AssertExpectations(t)
	})
}

func TestGetAllRunners(t *testing.T) {

	t.Run("Simple Case", func(t *testing.T) {
		logger, _ := logrusTest.NewNullLogger()
		mock := &mocks.GitLabClient{}
		mockListRunners(mock, 1)
		clinar := Clinar{Client: mock, Logger: logger}
		rners, err := clinar.GetAllRunners()
		require.NoError(t, err)
		assert.Len(t, rners, 10)
		mock.AssertExpectations(t)
	})

	t.Run("Multiple Calls", func(t *testing.T) {
		logger, _ := logrusTest.NewNullLogger()
		mock := &mocks.GitLabClient{}
		mockListRunners(mock, 10)
		clinar := Clinar{Client: mock, Logger: logger}
		rners, err := clinar.GetAllRunners()
		require.NoError(t, err)
		require.Len(t, rners, 100)
		assertRunnerIDContained(t, rners, 50)
		assertRunnerIDContained(t, rners, 75)
		assertRunnerIDContained(t, rners, 100)
		mock.AssertExpectations(t)
	})

	t.Run("Error at first call", func(t *testing.T) {
		logger, _ := logrusTest.NewNullLogger()
		mock := &mocks.GitLabClient{}
		mockListRunners(mock, 1, 1)
		clinar := Clinar{Client: mock, Logger: logger}
		rners, err := clinar.GetAllRunners()
		assert.Error(t, err)
		assert.Nil(t, rners)
		assert.EqualError(t, err, "Something went wrong 1")
		mock.AssertExpectations(t)
	})

	t.Run("Error at third call", func(t *testing.T) {
		logger, logHook := logrusTest.NewNullLogger()
		mock := &mocks.GitLabClient{}
		mockListRunners(mock, 10, 3)
		clinar := Clinar{Client: mock, Logger: logger}
		rners, err := clinar.GetAllRunners()
		require.NoError(t, err)
		assert.Len(t, rners, 90)
		assert.Len(t, logHook.Entries, 1)
		assert.Contains(t, logHook.Entries[0].Message, "Something went wrong 3")
		mock.AssertExpectations(t)
	})
}

func TestCleanupRunners(t *testing.T) {
	t.Run("Simple case", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		logger, logHook := logrusTest.NewNullLogger()
		mockDeleteRegisteredRunnerByID(mock, 5)
		clinar := Clinar{Client: mock, Logger: logger}
		clinar.CleanupRunners([]*gitlab.RunnerDetails{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})
		mock.AssertExpectations(t)
		assert.Len(t, logHook.Entries, 5)
		assert.Contains(t, logHook.Entries[0].Message, "Deleting")
		assert.Contains(t, logHook.Entries[1].Message, "Deleting")
		assert.Contains(t, logHook.Entries[2].Message, "Deleting")
		assert.Contains(t, logHook.Entries[3].Message, "Deleting")
		assert.Contains(t, logHook.Entries[4].Message, "Deleting")
		// Wait 50ms until goroutines are finished
	})

	t.Run("No runners to be purged", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		logger, logHook := logrusTest.NewNullLogger()
		clinar := Clinar{Client: mock, Logger: logger}
		clinar.CleanupRunners([]*gitlab.RunnerDetails{})
		mock.AssertExpectations(t)
		assert.Len(t, logHook.Entries, 1)
		assert.Equal(t, "No runners to be purged!", logHook.Entries[0].Message)
		assert.Equal(t, logrus.InfoLevel, logHook.Entries[0].Level)
	})

	t.Run("Error from DeleteRegisteredRunnerByID", func(t *testing.T) {
		mock := &mocks.GitLabClient{}
		logger, logHook := logrusTest.NewNullLogger()
		mock.EXPECT().DeleteRegisteredRunnerByID(123).Return(&gitlab.Response{Response: &http.Response{Status: "200 OK"}}, errors.New("Something went wrong"))
		clinar := Clinar{Client: mock, Logger: logger}
		clinar.CleanupRunners([]*gitlab.RunnerDetails{{ID: 123}})
		mock.AssertExpectations(t)
		assert.Len(t, logHook.Entries, 2)
		assert.Equal(t, "Deleting 123 - ", logHook.Entries[0].Message)
		assert.Equal(t, logrus.InfoLevel, logHook.Entries[0].Level)
		assert.Equal(t, "Something went wrong", logHook.Entries[1].Message)
		assert.Equal(t, logrus.ErrorLevel, logHook.Entries[1].Level)
	})

}

func mockGetRunnerDetails(mock *mocks.GitLabClient, numOfCalls int) {
	for i := 1; i <= numOfCalls; i++ {
		details := &gitlab.RunnerDetails{
			ID:   int(i),
			Name: fmt.Sprintf("someRunner%d", i),
			Projects: []struct {
				ID                int    "json:\"id\""
				Name              string "json:\"name\""
				NameWithNamespace string "json:\"name_with_namespace\""
				Path              string "json:\"path\""
				PathWithNamespace string "json:\"path_with_namespace\""
			}{
				{
					ID:   10 + i,
					Name: fmt.Sprintf("Project%d", i),
				},
			},
			Groups: []struct {
				ID     int    "json:\"id\""
				Name   string "json:\"name\""
				WebURL string "json:\"web_url\""
			}{
				{
					ID:   20 + i,
					Name: fmt.Sprintf("Group%d", i),
				},
			},
		}
		mock.EXPECT().GetRunnerDetails(i).Return(details, &gitlab.Response{TotalItems: 1, TotalPages: 1}, nil).Once()
	}
}

func mockListRunners(mock *mocks.GitLabClient, numOfCalls int, errorAt ...int) {
	for i := 1; i <= numOfCalls; i++ {
		opts := &gitlab.ListRunnersOptions{
			ListOptions: gitlab.ListOptions{
				PerPage: 100,
				Page:    i,
			},
			Status: gitlab.Ptr(runnerState),
		}
		baseId := 10 * i
		rners := []*gitlab.Runner{}
		for j := 0; j < 10; j++ {
			rner := &gitlab.Runner{
				ID: baseId + j,
			}
			rners = append(rners, rner)
		}

		resp := &gitlab.Response{
			TotalItems:   10 * numOfCalls,
			ItemsPerPage: 10,
			CurrentPage:  i,
			NextPage:     i + 1,
			TotalPages:   numOfCalls,
		}
		if shouldBeError(errorAt, i) {
			mock.EXPECT().ListRunners(opts).Return(nil, resp, fmt.Errorf("Something went wrong %d", i)).Once()
		} else {
			mock.EXPECT().ListRunners(opts).Return(rners, resp, nil).Once()
		}
	}
}

func shouldBeError(errorAt []int, pos int) bool {
	for _, posError := range errorAt {
		if posError == pos {
			return true
		}
	}
	return false
}

func mockDeleteRegisteredRunnerByID(mock *mocks.GitLabClient, numOfCalls int) {
	for i := 1; i <= numOfCalls; i++ {
		mock.EXPECT().DeleteRegisteredRunnerByID(i).Return(&gitlab.Response{Response: &http.Response{Status: "200 OK"}}, nil)
	}
}

func assertRunnerIDContained(t *testing.T, runners []*gitlab.Runner, id int) {
	for _, r := range runners {
		if r.ID == id {
			return
		}
	}
	t.Errorf("Runner with ID %d not found", id)
}
