package store

type IStore interface {
	Init() (IStore, error)
	Disconnect() error

	GetData() (interface{}, error)
	UpdateChecks() error
	//Save() error
	//Load() error
}
