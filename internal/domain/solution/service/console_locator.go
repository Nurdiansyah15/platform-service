// Package service berisi domain service solution.
package service

import (
	"context"
	"sync"

	solutionrepo "platform-service/internal/domain/solution/repository"
)

// ConsoleLocator menemukan ID solution console (CONSOLE_SOLUTION_CODE). ID di-cache setelah
// ditemukan; kegagalan tidak di-cache sehingga solution yang di-seed belakangan tetap terbaca.
type ConsoleLocator struct {
	solutions solutionrepo.SolutionRepository
	code      string

	mu sync.Mutex
	id string
}

func NewConsoleLocator(solutions solutionrepo.SolutionRepository, code string) *ConsoleLocator {
	return &ConsoleLocator{solutions: solutions, code: code}
}

func (l *ConsoleLocator) Code() string { return l.code }

// ConsoleSolutionID → CodeSolutionNotFound bila solution console belum di-seed.
func (l *ConsoleLocator) ConsoleSolutionID(ctx context.Context) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.id != "" {
		return l.id, nil
	}
	s, err := l.solutions.FindByCode(ctx, l.code)
	if err != nil {
		return "", err
	}
	l.id = s.ID
	return l.id, nil
}
