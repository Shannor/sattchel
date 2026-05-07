package tui

import (
	"fmt"
	"sync"
	"time"
)

// Spinner provides a simple terminal spinner animation.
type Spinner struct {
	mu     sync.Mutex
	done   chan struct{}
	active bool
	frame  int
	frames []string
}

// NewSpinner creates a spinner with default frames.
func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:   make(chan struct{}),
	}
}

// Start begins the spinner animation on stdout.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				if !s.active {
					s.mu.Unlock()
					return
				}
				frame := s.frames[s.frame%len(s.frames)]
				s.frame++
				s.mu.Unlock()
				fmt.Printf("\r%s ", frame)
			}
		}
	}()
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	close(s.done)
	fmt.Println()
}

// Update sets the spinner text alongside the animation.
func (s *Spinner) Update(msg string) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	frame := s.frames[s.frame%len(s.frames)]
	s.mu.Unlock()
	fmt.Printf("\r%s %s", frame, msg)
}
