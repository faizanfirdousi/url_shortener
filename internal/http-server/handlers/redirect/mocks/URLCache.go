package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

type URLCache struct {
	mock.Mock
}

func (m *URLCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *URLCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

type mockConstructorTestingTNewURLCache interface {
	mock.TestingT
	Cleanup(func())
}

func NewURLCache(t mockConstructorTestingTNewURLCache) *URLCache {
	mock := &URLCache{}
	mock.Mock.Test(t)
	t.Cleanup(func() { mock.AssertExpectations(t) })
	return mock
}
