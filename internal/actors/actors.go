package actors

type Actor interface {
	Act(msg string) error
}
