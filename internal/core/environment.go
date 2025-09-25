package core

// Environment indicates the runtime environment and controls logging/metrics behaviour.
type Environment string

const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)
