// Package ignoredpkg has four exported types — over a cap of 3 — but the
// package doc comment carries an ignore directive, so no diagnostic fires.
//
//plimsoll:ignore
package ignoredpkg

type A struct{}

type B struct{}

type C struct{}

type D struct{}
