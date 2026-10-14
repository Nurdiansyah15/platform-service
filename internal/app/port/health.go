package port

import "context"

type DatabaseHealth interface {
	Ping(ctx context.Context) error
}
