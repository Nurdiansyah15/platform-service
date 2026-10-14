package port

import "context"

// Transactor membungkus beberapa operasi repository dalam satu DB transaction.
// Repository yang dipanggil dengan ctx dari fn otomatis ikut transaksi.
type Transactor interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
