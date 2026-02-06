package health

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockTMDbClient is a mock TMDb client for testing
type MockTMDbClient struct {
	shouldFail bool
}

func (m *MockTMDbClient) Ping(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("TMDb connection failed")
	}
	return nil
}

// MockDoubanScraper is a mock Douban scraper for testing
type MockDoubanScraper struct {
	shouldFail bool
}

func (m *MockDoubanScraper) Ping(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("Douban connection failed")
	}
	return nil
}

// MockWikipediaClient is a mock Wikipedia client for testing
type MockWikipediaClient struct {
	shouldFail bool
}

func (m *MockWikipediaClient) Ping(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("Wikipedia connection failed")
	}
	return nil
}

// MockAIProvider is a mock AI provider for testing
type MockAIProvider struct {
	shouldFail bool
}

func (m *MockAIProvider) Ping(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("AI connection failed")
	}
	return nil
}

func TestServiceHealthChecker_CheckTMDb_Success(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{shouldFail: false},
		&MockDoubanScraper{},
		&MockWikipediaClient{},
		&MockAIProvider{},
	)

	err := checker.CheckTMDb(context.Background())
	assert.NoError(t, err)
}

func TestServiceHealthChecker_CheckTMDb_Failure(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{shouldFail: true},
		&MockDoubanScraper{},
		&MockWikipediaClient{},
		&MockAIProvider{},
	)

	err := checker.CheckTMDb(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TMDb")
}

func TestServiceHealthChecker_CheckDouban_Success(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{},
		&MockDoubanScraper{shouldFail: false},
		&MockWikipediaClient{},
		&MockAIProvider{},
	)

	err := checker.CheckDouban(context.Background())
	assert.NoError(t, err)
}

func TestServiceHealthChecker_CheckDouban_Failure(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{},
		&MockDoubanScraper{shouldFail: true},
		&MockWikipediaClient{},
		&MockAIProvider{},
	)

	err := checker.CheckDouban(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Douban")
}

func TestServiceHealthChecker_CheckWikipedia_Success(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{},
		&MockDoubanScraper{},
		&MockWikipediaClient{shouldFail: false},
		&MockAIProvider{},
	)

	err := checker.CheckWikipedia(context.Background())
	assert.NoError(t, err)
}

func TestServiceHealthChecker_CheckWikipedia_Failure(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{},
		&MockDoubanScraper{},
		&MockWikipediaClient{shouldFail: true},
		&MockAIProvider{},
	)

	err := checker.CheckWikipedia(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Wikipedia")
}

func TestServiceHealthChecker_CheckAI_Success(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{},
		&MockDoubanScraper{},
		&MockWikipediaClient{},
		&MockAIProvider{shouldFail: false},
	)

	err := checker.CheckAI(context.Background())
	assert.NoError(t, err)
}

func TestServiceHealthChecker_CheckAI_Failure(t *testing.T) {
	checker := NewServiceHealthChecker(
		&MockTMDbClient{},
		&MockDoubanScraper{},
		&MockWikipediaClient{},
		&MockAIProvider{shouldFail: true},
	)

	err := checker.CheckAI(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI")
}

func TestServiceHealthChecker_CheckTMDb_NilClient(t *testing.T) {
	checker := NewServiceHealthChecker(nil, nil, nil, nil)

	err := checker.CheckTMDb(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestServiceHealthChecker_CheckDouban_NilClient(t *testing.T) {
	checker := NewServiceHealthChecker(nil, nil, nil, nil)

	err := checker.CheckDouban(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestServiceHealthChecker_CheckWikipedia_NilClient(t *testing.T) {
	checker := NewServiceHealthChecker(nil, nil, nil, nil)

	err := checker.CheckWikipedia(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestServiceHealthChecker_CheckAI_NilClient(t *testing.T) {
	checker := NewServiceHealthChecker(nil, nil, nil, nil)

	err := checker.CheckAI(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}
