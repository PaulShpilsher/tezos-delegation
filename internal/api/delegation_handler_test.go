package api

import (
	"testing"
	"time"

	"tezos-delegation/internal/apperrors"
	"tezos-delegation/internal/mocks"
	"tezos-delegation/internal/model"

	"github.com/golang/mock/gomock"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestDelegationHandler_GetDelegations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockDelegationServicePort(ctrl)
	logger := zerolog.Nop()
	handler := NewDelegationHandler(service, logger)

	app := iris.New()
	app.Get("/xtz/delegations", handler.GetDelegations)
	test := httptest.New(t, app)

	expected := []model.Delegation{{TzktID: 1, Delegator: "tz1", Amount: 100, Level: 1, Timestamp: fixedTime()}}
	service.EXPECT().GetDelegations(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(expected, nil)

	resp := test.GET("/xtz/delegations").Expect().Status(200).JSON().Object()
	resp.Value("data").Array().Value(0).Object().HasValue("delegator", "tz1")
	resp.Value("data").Array().Value(0).Object().HasValue("amount", "100")
	resp.Value("data").Array().Value(0).Object().HasValue("timestamp", "2022-05-05T06:29:14Z")
}

func TestDelegationHandler_GetDelegations_OptionalQueryParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockDelegationServicePort(ctrl)
	logger := zerolog.Nop()
	handler := NewDelegationHandler(service, logger)

	app := iris.New()
	app.Get("/xtz/delegations", handler.GetDelegations)
	test := httptest.New(t, app)

	testCases := []struct {
		name     string
		query    string // no leading '?'
		page     int
		year     *int
		expected []model.Delegation
	}{
		{"default params", "", 1, nil, []model.Delegation{{TzktID: 1, Delegator: "tz1", Amount: 100, Level: 1, Timestamp: fixedTime()}}},
		{"page param", "page=2", 2, nil, []model.Delegation{{TzktID: 2, Delegator: "tz2", Amount: 200, Level: 2, Timestamp: fixedTime()}}},
		{"year param", "year=2022", 1, intPtr(2022), []model.Delegation{{TzktID: 3, Delegator: "tz3", Amount: 300, Level: 3, Timestamp: fixedTime()}}},
		{"page and year", "page=3&year=2021", 3, intPtr(2021), []model.Delegation{{TzktID: 4, Delegator: "tz4", Amount: 400, Level: 4, Timestamp: fixedTime()}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service.EXPECT().GetDelegations(gomock.Any(), tc.page, gomock.Any(), tc.year).Return(tc.expected, nil)
			resp := test.GET("/xtz/delegations").WithQueryString(tc.query).Expect().Status(200).JSON().Object()
			if len(tc.expected) > 0 {
				resp.Value("data").Array().Value(0).Object().HasValue("delegator", tc.expected[0].Delegator)
			} else {
				resp.Value("data").Array().IsEmpty()
			}
		})
	}
}

func TestDelegationHandler_GetDelegations_ErrorConditions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockDelegationServicePort(ctrl)
	logger := zerolog.Nop()
	handler := NewDelegationHandler(service, logger)

	app := iris.New()
	app.Get("/xtz/delegations", handler.GetDelegations)
	test := httptest.New(t, app)

	t.Run("invalid page", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("page=abc").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid page parameter: must be a positive integer")
	})
	t.Run("invalid year", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("year=bad").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid year parameter: must be a valid year from 2018 onwards")
	})
	t.Run("negative page", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("page=-1").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid page parameter: must be a positive integer")
	})
	t.Run("negative year", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("year=-5").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid year parameter: must be a valid year from 2018 onwards")
	})
	t.Run("year before 2018", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("year=2017").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid year parameter: must be a valid year from 2018 onwards")
	})
	t.Run("invalid pageSize", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("pageSize=abc").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid pageSize parameter: must be between 1 and 1000")
	})
	t.Run("pageSize too large", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("pageSize=1001").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid pageSize parameter: must be between 1 and 1000")
	})
	t.Run("pageSize zero", func(t *testing.T) {
		resp := test.GET("/xtz/delegations").WithQueryString("pageSize=0").Expect().Status(400).JSON().Object()
		resp.Value("error").String().IsEqual("Invalid pageSize parameter: must be between 1 and 1000")
	})
	t.Run("service database error", func(t *testing.T) {
		dbErr := apperrors.NewDatabaseError("query", "connection failed")
		service.EXPECT().GetDelegations(gomock.Any(), 1, gomock.Any(), gomock.Any()).Return(nil, dbErr)
		resp := test.GET("/xtz/delegations").Expect().Status(500).JSON().Object()
		resp.Value("error").String().IsEqual("Database error")
	})
	t.Run("service general error", func(t *testing.T) {
		service.EXPECT().GetDelegations(gomock.Any(), 1, gomock.Any(), gomock.Any()).Return(nil, assert.AnError)
		resp := test.GET("/xtz/delegations").Expect().Status(500).JSON().Object()
		resp.Value("error").String().IsEqual("Internal server error")
	})
}

func TestDelegationHandler_GetDelegations_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockDelegationServicePort(ctrl)
	logger := zerolog.Nop()
	handler := NewDelegationHandler(service, logger)

	app := iris.New()
	app.Get("/xtz/delegations", handler.GetDelegations)
	test := httptest.New(t, app)

	t.Run("empty result", func(t *testing.T) {
		service.EXPECT().GetDelegations(gomock.Any(), 1, gomock.Any(), gomock.Any()).Return([]model.Delegation{}, nil)
		resp := test.GET("/xtz/delegations").Expect().Status(200).JSON().Object()
		resp.Value("data").Array().IsEmpty()
	})

	t.Run("large page number", func(t *testing.T) {
		service.EXPECT().GetDelegations(gomock.Any(), 9999, gomock.Any(), gomock.Any()).Return([]model.Delegation{}, nil)
		resp := test.GET("/xtz/delegations").WithQueryString("page=9999").Expect().Status(200).JSON().Object()
		resp.Value("data").Array().IsEmpty()
	})

	t.Run("year with no data", func(t *testing.T) {
		year := 2019
		service.EXPECT().GetDelegations(gomock.Any(), 1, gomock.Any(), &year).Return([]model.Delegation{}, nil)
		resp := test.GET("/xtz/delegations").WithQueryString("year=2019").Expect().Status(200).JSON().Object()
		resp.Value("data").Array().IsEmpty()
	})

	t.Run("valid year 2018", func(t *testing.T) {
		year := 2018
		expected := []model.Delegation{{TzktID: 1, Delegator: "tz1", Amount: 100, Level: 1, Timestamp: fixedTime()}}
		service.EXPECT().GetDelegations(gomock.Any(), 1, gomock.Any(), &year).Return(expected, nil)
		resp := test.GET("/xtz/delegations").WithQueryString("year=2018").Expect().Status(200).JSON().Object()
		resp.Value("data").Array().Value(0).Object().HasValue("delegator", "tz1")
	})

	t.Run("valid year 2023", func(t *testing.T) {
		year := 2023
		expected := []model.Delegation{{TzktID: 2, Delegator: "tz2", Amount: 200, Level: 2, Timestamp: fixedTime()}}
		service.EXPECT().GetDelegations(gomock.Any(), 1, gomock.Any(), &year).Return(expected, nil)
		resp := test.GET("/xtz/delegations").WithQueryString("year=2023").Expect().Status(200).JSON().Object()
		resp.Value("data").Array().Value(0).Object().HasValue("delegator", "tz2")
	})
}

func TestDelegationHandler_GetDelegations_ValidationScenarios(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := mocks.NewMockDelegationServicePort(ctrl)
	logger := zerolog.Nop()
	handler := NewDelegationHandler(service, logger)

	app := iris.New()
	app.Get("/xtz/delegations", handler.GetDelegations)
	test := httptest.New(t, app)

	t.Run("page parameter edge cases", func(t *testing.T) {
		testCases := []struct {
			query       string
			expectedMsg string
		}{
			{"page=0", "Invalid page parameter: must be a positive integer"},
			{"page=-1", "Invalid page parameter: must be a positive integer"},
			{"page=abc", "Invalid page parameter: must be a positive integer"},
			{"page=", "Invalid page parameter: must be a positive integer"},
		}

		for _, tc := range testCases {
			t.Run(tc.query, func(t *testing.T) {
				resp := test.GET("/xtz/delegations").WithQueryString(tc.query).Expect().Status(400).JSON().Object()
				resp.Value("error").String().IsEqual(tc.expectedMsg)
			})
		}
	})

	t.Run("pageSize parameter edge cases", func(t *testing.T) {
		testCases := []struct {
			query       string
			expectedMsg string
		}{
			{"pageSize=0", "Invalid pageSize parameter: must be between 1 and 1000"},
			{"pageSize=-1", "Invalid pageSize parameter: must be between 1 and 1000"},
			{"pageSize=1001", "Invalid pageSize parameter: must be between 1 and 1000"},
			{"pageSize=abc", "Invalid pageSize parameter: must be between 1 and 1000"},
			{"pageSize=", "Invalid pageSize parameter: must be between 1 and 1000"},
		}

		for _, tc := range testCases {
			t.Run(tc.query, func(t *testing.T) {
				resp := test.GET("/xtz/delegations").WithQueryString(tc.query).Expect().Status(400).JSON().Object()
				resp.Value("error").String().IsEqual(tc.expectedMsg)
			})
		}
	})

	t.Run("year parameter edge cases", func(t *testing.T) {
		testCases := []struct {
			query       string
			expectedMsg string
		}{
			{"year=2017", "Invalid year parameter: must be a valid year from 2018 onwards"},
			{"year=2016", "Invalid year parameter: must be a valid year from 2018 onwards"},
			{"year=abc", "Invalid year parameter: must be a valid year from 2018 onwards"},
		}

		for _, tc := range testCases {
			t.Run(tc.query, func(t *testing.T) {
				resp := test.GET("/xtz/delegations").WithQueryString(tc.query).Expect().Status(400).JSON().Object()
				resp.Value("error").String().IsEqual(tc.expectedMsg)
			})
		}
	})

	t.Run("valid year parameter cases", func(t *testing.T) {
		// Test that empty year parameter is valid (no year filter)
		expected := []model.Delegation{{TzktID: 1, Delegator: "tz1", Amount: 100, Level: 1, Timestamp: fixedTime()}}
		service.EXPECT().GetDelegations(gomock.Any(), 1, gomock.Any(), (*int)(nil)).Return(expected, nil)

		resp := test.GET("/xtz/delegations").WithQueryString("year=").Expect().Status(200).JSON().Object()
		resp.Value("data").Array().Value(0).Object().HasValue("delegator", "tz1")
	})
}

func intPtr(i int) *int { return &i }

func fixedTime() time.Time {
	return time.Date(2022, 5, 5, 6, 29, 14, 0, time.UTC)
}
