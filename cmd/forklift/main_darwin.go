//go:build darwin

package main

// This is needed because Forklift imports Docker Compose, which on darwin imports
// github.com/fsnotify/fsevents, which requires CGo; and we need to make CGo do something so that
// we can generate CGo runtime stuff for the linker:
import "C"
