package model

// RecommendationSeverity indicates the urgency level of a recommendation.
type RecommendationSeverity int

const (
	SeverityNormal   RecommendationSeverity = iota
	SeverityWarning
	SeverityCritical
)

// RecommendationCategory groups related recommendations.
type RecommendationCategory int

const (
	CategoryResourcePressure RecommendationCategory = iota
	CategoryShardHealth
	CategoryIndexConfig
	CategoryHotspot
	CategoryIndexLifecycle
)

// Recommendation is a single actionable suggestion derived from cluster state.
type Recommendation struct {
	Severity RecommendationSeverity
	Category RecommendationCategory
	Title    string
	Detail   string
}
