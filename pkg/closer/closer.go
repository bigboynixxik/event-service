package closer

import (
	"context"
	"fmt"
	"sync"
)

type Func func(ctx context.Context) error

type Closer struct {
	mu    sync.Mutex
	funcs []Func
}

func New() *Closer {
	return &Closer{}
}

func (c *Closer) Add(f Func) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.funcs = append(c.funcs, f)
}

func (c *Closer) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error

	for _, f := range c.funcs {
		if err := f(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("closer errors: %v", errs)
	}
	return nil
}
