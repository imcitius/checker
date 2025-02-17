package store

type IStore interface {
	Init(dbConfig DBConfig) (IStore, error)
	Disconnect() error
}
