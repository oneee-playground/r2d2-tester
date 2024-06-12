package job

import "github.com/google/uuid"

type SectionType string

const (
	TypeScenario SectionType = "SCENARIO"
	TypeLoad     SectionType = "LOAD"
)

type Resource struct {
	Image     string  `json:"image"`
	Name      string  `json:"name"`
	Port      uint16  `json:"port"`
	CPU       float64 `json:"cpu"`
	Memory    uint64  `json:"memory"`
	IsPrimary bool    `json:"isPrimary"`
}

type Section struct {
	ID   uuid.UUID   `json:"id"`
	Type SectionType `json:"type"`
}

type Submission struct {
	ID         uuid.UUID `json:"id"`
	Repository string    `json:"repositoy"`
	CommitHash string    `json:"commitHash"`
}

type Job struct {
	TaskID uuid.UUID `json:"taskID"`

	Resources []Resource `json:"resources"`
	Sections  []Section  `json:"sections"`

	Submission Submission `json:"submission"`
}
