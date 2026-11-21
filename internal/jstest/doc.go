// Package jstest runs ui's client script modules in go test, through a
// pure-Go JavaScript engine (ADR 0006). It is a module of its own so the
// engine never enters ui's go.mod: only this test imports it.
package jstest
