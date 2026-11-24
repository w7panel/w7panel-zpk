package storage

import (
	"context"
	"io"
)

type Store interface {
	io.Closer
	Put(context.Context, []byte) (string, error)
	Open(context.Context, string) (io.ReadCloser, error)
}
