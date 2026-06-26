package widepkg // want `package widepkg has 4 exported types, over the load line of 3`

// Four exported types; the test runs with maxExportedTypes=3, so the package
// load line trips. The unexported type does not count.

type Alpha struct{}

type Beta struct{}

type Gamma struct{}

type Delta struct{}

type unexported struct{}

// Aliases and non-struct named types count too — they widen the exported
// surface just like a struct. (This file already trips at 4; these would push
// it higher, but we keep the fixture minimal and put them in a second file.)
