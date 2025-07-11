package services

import (
	"context"
	"testing"
	"time"

	"tezos-delegation/internal/mocks"
	"tezos-delegation/internal/model"

	"tezos-delegation/internal/db"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestDelegationService_GetDelegations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	service := NewDelegationService(repo, logger)

	ctx := context.Background()
	pageNo, pageSize := 1, 10
	var year *int = nil

	expected := []model.Delegation{{TzktID: 1, Delegator: "tz1", Amount: 100, Level: 1, Timestamp: fixedTime()}}
	repo.EXPECT().ListDelegations(ctx, pageSize, 0, year).Return(expected, nil)

	result, err := service.GetDelegations(ctx, pageNo, pageSize, year)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestDelegationService_GetDelegations_InvalidPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	service := NewDelegationService(repo, logger)
	ctx := context.Background()
	var year *int = nil

	cases := []struct {
		name   string
		pageNo int
		pageSz int
	}{
		{"pageNo < 1", 0, 10},
		{"pageSize < 1", 1, 0},
		{"pageSize > 1000", 1, 1001},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := service.GetDelegations(ctx, c.pageNo, c.pageSz, year)
			assert.Error(t, err)
			assert.True(t, err != nil && err.Error() != "", "should return a validation error")
		})
	}
}

func TestDelegationService_GetDelegations_InvalidYear(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	service := NewDelegationService(repo, logger)
	ctx := context.Background()
	pageNo, pageSize := 1, 10

	badYears := []int{2017, 2101}
	for _, y := range badYears {
		year := y
		t.Run("year invalid", func(t *testing.T) {
			_, err := service.GetDelegations(ctx, pageNo, pageSize, &year)
			assert.Error(t, err)
			assert.True(t, err != nil && err.Error() != "", "should return a validation error")
		})
	}
}

func TestDelegationService_GetDelegations_NoDelegations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	service := NewDelegationService(repo, logger)
	ctx := context.Background()
	pageNo, pageSize := 1, 10
	var year *int = nil

	repo.EXPECT().ListDelegations(ctx, pageSize, 0, year).Return(nil, db.ErrNoDelegations)

	result, err := service.GetDelegations(ctx, pageNo, pageSize, year)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestDelegationService_GetDelegations_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	service := NewDelegationService(repo, logger)
	ctx := context.Background()
	pageNo, pageSize := 1, 10
	var year *int = nil

	repo.EXPECT().ListDelegations(ctx, pageSize, 0, year).Return(nil, assert.AnError)

	result, err := service.GetDelegations(ctx, pageNo, pageSize, year)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDelegationService_GetDelegations_ValidYear(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	service := NewDelegationService(repo, logger)
	ctx := context.Background()
	pageNo, pageSize := 1, 10
	year := 2022

	expected := []model.Delegation{{TzktID: 2, Delegator: "tz2", Amount: 200, Level: 2, Timestamp: fixedTime()}}
	repo.EXPECT().ListDelegations(ctx, pageSize, 0, &year).Return(expected, nil)

	result, err := service.GetDelegations(ctx, pageNo, pageSize, &year)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestDelegationService_GetDelegations_PaginationOffset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockDelegationRepositoryPort(ctrl)
	logger := zerolog.Nop()
	service := NewDelegationService(repo, logger)
	ctx := context.Background()
	pageNo, pageSize := 3, 5
	var year *int = nil

	expected := []model.Delegation{{TzktID: 3, Delegator: "tz3", Amount: 300, Level: 3, Timestamp: fixedTime()}}
	repo.EXPECT().ListDelegations(ctx, pageSize, 10, year).Return(expected, nil)

	result, err := service.GetDelegations(ctx, pageNo, pageSize, year)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func fixedTime() time.Time {
	return time.Date(2022, 5, 5, 6, 29, 14, 0, time.UTC)
}
