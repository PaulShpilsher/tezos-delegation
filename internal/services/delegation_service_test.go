package services

import (
	"context"
	"testing"
	"time"

	"tezos-delegation/internal/mocks"
	"tezos-delegation/internal/model"

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

func fixedTime() time.Time {
	return time.Date(2022, 5, 5, 6, 29, 14, 0, time.UTC)
}
