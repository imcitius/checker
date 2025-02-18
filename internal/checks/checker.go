package checks

// Checker is an interface that all health checks should implement.
// It defines a universal Run method.
type Checker interface {
	// Run executes the health check and returns:
	// - a bool indicating if the check passed, and
	// - a message detailing the result.
	Run() (bool, string)
} 