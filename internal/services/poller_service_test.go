package services

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"tezos-delegation/internal/mocks"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestPollerService_syncDelegationsBatch_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	ps := &PollerService{
		repo:   repo,
		logger: logger,
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`[{"id":1,"timestamp":"2022-05-05T06:29:14Z","amount":100,"sender":{"address":"tz1"},"level":1}]`)),
				Header:     make(http.Header),
			}
		})},
	}

	ctx := context.Background()
	repo.EXPECT().GetLatestTzktID(ctx).Return(int64(0), nil)
	repo.EXPECT().InsertDelegations(gomock.Any()).Return(nil)

	caughtUp, err := ps.syncDelegationsBatch(ctx)
	assert.NoError(t, err)
	assert.True(t, caughtUp)
}

func TestPollerService_syncDelegationsBatch_NoNewData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	ps := &PollerService{
		repo:   repo,
		logger: logger,
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`[]`)),
				Header:     make(http.Header),
			}
		})},
	}

	ctx := context.Background()
	repo.EXPECT().GetLatestTzktID(ctx).Return(int64(0), nil)

	caughtUp, err := ps.syncDelegationsBatch(ctx)
	assert.NoError(t, err)
	assert.True(t, caughtUp)
}

func TestPollerService_syncDelegationsBatch_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	ps := &PollerService{
		repo:   repo,
		logger: logger,
	}

	ctx := context.Background()
	repo.EXPECT().GetLatestTzktID(ctx).Return(int64(0), errors.New("db error"))

	_, err := ps.syncDelegationsBatch(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get latest TzktID")
}

func TestPollerService_syncDelegationsBatch_APIError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	ps := &PollerService{
		repo:   repo,
		logger: logger,
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: 500,
				Body:       io.NopCloser(strings.NewReader(`error`)),
				Header:     make(http.Header),
			}
		})},
	}

	ctx := context.Background()
	repo.EXPECT().GetLatestTzktID(ctx).Return(int64(0), nil)

	_, err := ps.syncDelegationsBatch(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch delegations from Tzkt API")
}

func TestPollerService_syncDelegationsBatch_ContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	ps := &PollerService{
		repo:   repo,
		logger: logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ps.syncDelegationsBatch(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context cancelled")
}
