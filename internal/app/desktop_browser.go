//go:build !desktop || (!darwin && !windows)

package app

func cmdDesktop() { cmdWeb() }
