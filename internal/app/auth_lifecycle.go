package app

import "sync"

// Serializes credential commits with reset. Work started before a reset may
// finish, but must never restore credentials or publish health for a new session.
var authLifecycle struct {
	sync.Mutex
	generation uint64
}

func authGeneration() uint64 {
	authLifecycle.Lock()
	defer authLifecycle.Unlock()
	return authLifecycle.generation
}

func withAuthGeneration(generation uint64, fn func()) bool {
	authLifecycle.Lock()
	defer authLifecycle.Unlock()
	if generation != authLifecycle.generation {
		return false
	}
	fn()
	return true
}
