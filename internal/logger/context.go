package logger

import "context"

type ctxKeyRequestID struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID{}, id)
}

func RequestIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxKeyRequestID{}).(string)
	return id, ok
}
