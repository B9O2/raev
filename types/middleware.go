package types

type ClassMiddleware interface {
	Handle(*Class)
}
