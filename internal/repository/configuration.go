package repository

type Configuration[T any] interface {
	SetConfig(config T) error
	GetConfig() (*T, error)
}
