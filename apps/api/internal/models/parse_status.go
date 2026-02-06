package models

import (
	"time"
)

// ParseProgress holds the full progress state for a parse operation
type ParseProgress struct {
	TaskID      string       `json:"taskId"`
	Filename    string       `json:"filename"`
	Status      ParseStatus  `json:"status"`
	Steps       []ParseStep  `json:"steps"`
	CurrentStep int          `json:"currentStep"`
	Percentage  int          `json:"percentage"`
	Message     string       `json:"message,omitempty"`
	Result      *ParseResult `json:"result,omitempty"`
	StartedAt   time.Time    `json:"startedAt"`
	CompletedAt *time.Time   `json:"completedAt,omitempty"`
}

// ParseStep represents a single step in the parse process
type ParseStep struct {
	Name      string     `json:"name"`
	Label     string     `json:"label"`
	Status    StepStatus `json:"status"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// StepStatus represents the status of a single step
type StepStatus string

const (
	StepPending    StepStatus = "pending"
	StepInProgress StepStatus = "in_progress"
	StepSuccess    StepStatus = "success"
	StepFailed     StepStatus = "failed"
	StepSkipped    StepStatus = "skipped"
)

// ParseResult represents the result of a successful parse operation
type ParseResult struct {
	MediaID        string         `json:"mediaId,omitempty"`
	Title          string         `json:"title,omitempty"`
	Year           int            `json:"year,omitempty"`
	MediaType      string         `json:"mediaType,omitempty"`
	MetadataSource MetadataSource `json:"metadataSource,omitempty"`
	Confidence     float64        `json:"confidence,omitempty"`
}

// StandardParseSteps returns the standard steps for a parse operation
func StandardParseSteps() []ParseStep {
	return []ParseStep{
		{Name: "filename_extract", Label: "解析檔名", Status: StepPending},
		{Name: "tmdb_search", Label: "搜尋 TMDb", Status: StepPending},
		{Name: "douban_search", Label: "搜尋豆瓣", Status: StepPending},
		{Name: "wikipedia_search", Label: "搜尋 Wikipedia", Status: StepPending},
		{Name: "ai_retry", Label: "AI 重試", Status: StepPending},
		{Name: "download_poster", Label: "下載海報", Status: StepPending},
	}
}

// NewParseProgress creates a new ParseProgress with standard steps
func NewParseProgress(taskID, filename string) *ParseProgress {
	return &ParseProgress{
		TaskID:      taskID,
		Filename:    filename,
		Status:      ParseStatusPending,
		Steps:       StandardParseSteps(),
		CurrentStep: 0,
		Percentage:  0,
		StartedAt:   time.Now(),
	}
}

// StartStep marks a step as in progress and updates the current step
func (p *ParseProgress) StartStep(stepIndex int) {
	if stepIndex < 0 || stepIndex >= len(p.Steps) {
		return
	}

	now := time.Now()
	p.Steps[stepIndex].Status = StepInProgress
	p.Steps[stepIndex].StartedAt = &now
	p.CurrentStep = stepIndex
	p.Status = ParseStatusParsing
	p.updatePercentage()
}

// CompleteStep marks a step as successful
func (p *ParseProgress) CompleteStep(stepIndex int) {
	if stepIndex < 0 || stepIndex >= len(p.Steps) {
		return
	}

	now := time.Now()
	p.Steps[stepIndex].Status = StepSuccess
	p.Steps[stepIndex].EndedAt = &now
	p.updatePercentage()
}

// FailStep marks a step as failed with an error message
func (p *ParseProgress) FailStep(stepIndex int, errorMsg string) {
	if stepIndex < 0 || stepIndex >= len(p.Steps) {
		return
	}

	now := time.Now()
	p.Steps[stepIndex].Status = StepFailed
	p.Steps[stepIndex].EndedAt = &now
	p.Steps[stepIndex].Error = errorMsg
	p.updatePercentage()
}

// SkipStep marks a step as skipped
func (p *ParseProgress) SkipStep(stepIndex int) {
	if stepIndex < 0 || stepIndex >= len(p.Steps) {
		return
	}

	now := time.Now()
	p.Steps[stepIndex].Status = StepSkipped
	p.Steps[stepIndex].EndedAt = &now
	p.updatePercentage()
}

// Complete marks the entire parse operation as complete
func (p *ParseProgress) Complete(result *ParseResult) {
	now := time.Now()
	p.Status = ParseStatusSuccess
	p.CompletedAt = &now
	p.Result = result
	p.Percentage = 100
}

// CompleteWithWarning marks the parse as complete but with warnings
func (p *ParseProgress) CompleteWithWarning(message string) {
	now := time.Now()
	p.Status = ParseStatusNeedsAI
	p.CompletedAt = &now
	p.Message = message
	p.updatePercentage()
}

// Fail marks the entire parse operation as failed
func (p *ParseProgress) Fail(message string) {
	now := time.Now()
	p.Status = ParseStatusFailed
	p.CompletedAt = &now
	p.Message = message
}

// updatePercentage calculates the percentage based on completed steps
func (p *ParseProgress) updatePercentage() {
	if len(p.Steps) == 0 {
		p.Percentage = 0
		return
	}

	completedCount := 0
	for _, step := range p.Steps {
		if step.Status == StepSuccess || step.Status == StepSkipped {
			completedCount++
		}
	}

	p.Percentage = (completedCount * 100) / len(p.Steps)
}

// GetStepByName returns a step by its name
func (p *ParseProgress) GetStepByName(name string) *ParseStep {
	for i := range p.Steps {
		if p.Steps[i].Name == name {
			return &p.Steps[i]
		}
	}
	return nil
}

// GetStepIndex returns the index of a step by its name
func (p *ParseProgress) GetStepIndex(name string) int {
	for i, step := range p.Steps {
		if step.Name == name {
			return i
		}
	}
	return -1
}

// HasFailedSteps returns true if any step has failed
func (p *ParseProgress) HasFailedSteps() bool {
	for _, step := range p.Steps {
		if step.Status == StepFailed {
			return true
		}
	}
	return false
}

// GetFailedSteps returns all failed steps
func (p *ParseProgress) GetFailedSteps() []ParseStep {
	failed := make([]ParseStep, 0)
	for _, step := range p.Steps {
		if step.Status == StepFailed {
			failed = append(failed, step)
		}
	}
	return failed
}

// IsComplete returns true if the parse operation is complete (success, failed, or needs_ai)
func (p *ParseProgress) IsComplete() bool {
	return p.Status == ParseStatusSuccess || p.Status == ParseStatusFailed || p.Status == ParseStatusNeedsAI
}
