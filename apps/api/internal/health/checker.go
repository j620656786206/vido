package health

import (
	"context"
	"errors"
)

// Pingable defines the interface for services that can be health checked
type Pingable interface {
	Ping(ctx context.Context) error
}

// ServiceHealthChecker implements HealthChecker for actual service clients
type ServiceHealthChecker struct {
	tmdb      Pingable
	douban    Pingable
	wikipedia Pingable
	ai        Pingable
}

// NewServiceHealthChecker creates a new ServiceHealthChecker
func NewServiceHealthChecker(tmdb, douban, wikipedia, ai Pingable) *ServiceHealthChecker {
	return &ServiceHealthChecker{
		tmdb:      tmdb,
		douban:    douban,
		wikipedia: wikipedia,
		ai:        ai,
	}
}

// CheckTMDb checks the health of TMDb API
func (c *ServiceHealthChecker) CheckTMDb(ctx context.Context) error {
	if c.tmdb == nil {
		return errors.New("TMDb client not configured")
	}
	return c.tmdb.Ping(ctx)
}

// CheckDouban checks the health of Douban scraper
func (c *ServiceHealthChecker) CheckDouban(ctx context.Context) error {
	if c.douban == nil {
		return errors.New("Douban scraper not configured")
	}
	return c.douban.Ping(ctx)
}

// CheckWikipedia checks the health of Wikipedia API
func (c *ServiceHealthChecker) CheckWikipedia(ctx context.Context) error {
	if c.wikipedia == nil {
		return errors.New("Wikipedia client not configured")
	}
	return c.wikipedia.Ping(ctx)
}

// CheckAI checks the health of AI provider
func (c *ServiceHealthChecker) CheckAI(ctx context.Context) error {
	if c.ai == nil {
		return errors.New("AI provider not configured")
	}
	return c.ai.Ping(ctx)
}
