package problem

import (
	"context"

	openapi "github.com/Dynatrace/dynatrace-operator/pkg/clients/generated"
	"github.com/pkg/errors"
)

type APIClient interface {
	List(ctx context.Context, from, to string) (*openapi.Problems, error)
	Get(ctx context.Context, id string) (*Problem, error)
}

type Client struct {
	api openapi.ProblemsAPI
}

func NewClient(openApiClient openapi.ProblemsAPI) *Client {
	return &Client{
		api: openApiClient,
	}
}

func (c *Client) List(ctx context.Context, from, to string) (*openapi.Problems, error) {
	result, _, err := c.api.GetProblems(ctx).From(from).To(to).Execute()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list the problems")
	}

	return result, nil
}

// As alternative to generated return models,
// its possible to use custom structs to filter out unused fields of the generated models
type Problem struct {
	Status string
	Title  string
}

func (c *Client) Get(ctx context.Context, id string) (*Problem, error) {
	result, _, err := c.api.GetProblem(ctx, id).Execute()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the problem: "+id)
	}

	return &Problem{
		Status: result.Status,
		Title:  result.Title,
	}, nil
}
