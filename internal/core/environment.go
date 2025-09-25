package core

// Environment represents the deployment environment of the service.
type Environment string

const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Testing     Environment = "testing"
	Production  Environment = "production"
)

// String returns the string representation of the environment.
func (e Environment) String() string {
	return string(e)
}

// IsProduction reports whether the environment corresponds to production.
func (e Environment) IsProduction() bool {
	return e == Production
}

// ParseEnvironment normalises the provided value into one of the known environments.
// Unknown values fall back to Development so the application can still start
// with sensible defaults.
func ParseEnvironment(v string) Environment {
	switch Environment(v) {
	case Production:
		return Production
	case Staging:
		return Staging
	case Testing:
		return Testing
	default:
		return Development
	}
}
