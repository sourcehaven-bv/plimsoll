// Package raisedpkg has four exported types but raises its own load line to 10
// via a package directive, so it stays under and no diagnostic fires.
//
//plimsoll:max-exported-types=10
package raisedpkg

type A struct{}

type B struct{}

type C struct{}

type D struct{}
