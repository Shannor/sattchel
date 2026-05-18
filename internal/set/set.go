// Package set provides a generic Set data structure backed by a map for O(1) lookups.
package set

// Set is a generic set backed by a map for O(1) lookups.
type Set[T comparable] struct {
	m map[T]struct{}
}

// New creates an empty Set.
func New[T comparable]() Set[T] {
	return Set[T]{m: make(map[T]struct{})}
}

// NewFrom creates a Set from a slice of items.
func NewFrom[T comparable](items []T) Set[T] {
	m := make(map[T]struct{}, len(items))
	for _, item := range items {
		m[item] = struct{}{}
	}
	return Set[T]{m: m}
}

// NewFromFunc creates a Set from a slice of items using a key extraction function.
// The items themselves don't need to be comparable; only the key returned by keyFunc must be.
func NewFromFunc[K comparable, V any](items []V, keyFunc func(V) K) Set[K] {
	m := make(map[K]struct{}, len(items))
	for _, item := range items {
		m[keyFunc(item)] = struct{}{}
	}
	return Set[K]{m: m}
}

// Contains reports whether item is in the set.
func (s Set[T]) Contains(item T) bool {
	_, ok := s.m[item]
	return ok
}

// Add inserts item into the set.
func (s *Set[T]) Add(item T) {
	s.m[item] = struct{}{}
}

// Remove removes item from the set.
func (s *Set[T]) Remove(item T) {
	delete(s.m, item)
}

// Len returns the number of items in the set.
func (s Set[T]) Len() int {
	return len(s.m)
}

// Clear removes all items from the set.
func (s *Set[T]) Clear() {
	for k := range s.m {
		delete(s.m, k)
	}
}

// Union returns a new Set containing all items from both sets.
func (s Set[T]) Union(other Set[T]) Set[T] {
	result := New[T]()
	for item := range s.m {
		result.Add(item)
	}
	for item := range other.m {
		result.Add(item)
	}
	return result
}

// Difference returns a new Set containing items in s but not in other.
func (s Set[T]) Difference(other Set[T]) Set[T] {
	result := New[T]()
	for item := range s.m {
		if !other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// Intersection returns a new Set containing items present in both sets.
func (s Set[T]) Intersection(other Set[T]) Set[T] {
	result := New[T]()
	for item := range s.m {
		if other.Contains(item) {
			result.Add(item)
		}
	}
	return result
}

// Items returns all items in the set as a slice.
func (s Set[T]) Items() []T {
	items := make([]T, 0, len(s.m))
	for item := range s.m {
		items = append(items, item)
	}
	return items
}
