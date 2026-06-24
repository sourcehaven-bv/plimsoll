package basic

// Small is well under the caps — no diagnostic.
type Small struct {
	A int
	B int
}

func (Small) M1() {}
func (Small) M2() {}

// TooManyMethods has 4 methods, all exported; the test runs with maxMethods=3
// and maxExportedMethods=3, so both lines trip.
type TooManyMethods struct{} // want `type TooManyMethods has 4 methods, over the load line of 3` `type TooManyMethods has 4 exported methods, over the load line of 3`

func (TooManyMethods) A() {}
func (TooManyMethods) B() {}
func (TooManyMethods) C() {}
func (TooManyMethods) D() {}

// PointerRecv mixes pointer and value receivers; both count. 4 total, all
// exported, so both the total and exported lines trip.
type PointerRecv struct{} // want `type PointerRecv has 4 methods, over the load line of 3` `type PointerRecv has 4 exported methods, over the load line of 3`

func (PointerRecv) A()  {}
func (*PointerRecv) B() {}
func (PointerRecv) C()  {}
func (*PointerRecv) D() {}

// WideStruct has 4 exported fields; the test runs with maxFields=3. Unexported
// fields do not count.
type WideStruct struct { // want `struct WideStruct has 4 exported fields, over the load line of 3`
	A int
	B int
	C int
	D int
	e int
}

// Ignored exceeds both caps but is annotated to skip.
//
//plimsoll:ignore
type Ignored struct {
	A int
	B int
	C int
	D int
}

func (Ignored) A1() {}
func (Ignored) A2() {}
func (Ignored) A3() {}
func (Ignored) A4() {}

// Raised has 5 methods, all exported, and raises both the total line (to 10)
// and the exported line (to 10), so neither trips.
//
//plimsoll:max-methods=10
//plimsoll:max-exported-methods=10
type Raised struct{}

func (Raised) A() {}
func (Raised) B() {}
func (Raised) C() {}
func (Raised) D() {}
func (Raised) E() {}

// ManyPrivate has 4 total methods (trips the total cap of 3) but only 1
// exported, so it does NOT trip the exported cap of 3. This is the whole point
// of the exported dimension: private helpers are not a god-object signal.
type ManyPrivate struct{} // want `type ManyPrivate has 4 methods, over the load line of 3`

func (ManyPrivate) Pub() {}
func (ManyPrivate) a()   {}
func (ManyPrivate) b()   {}
func (ManyPrivate) c()   {}

// WidePublicAPI has 4 exported methods; the test runs with maxExportedMethods=3.
// It also trips the total cap (4 > 3), so both diagnostics fire.
type WidePublicAPI struct{} // want `type WidePublicAPI has 4 methods, over the load line of 3` `type WidePublicAPI has 4 exported methods, over the load line of 3`

func (WidePublicAPI) A() {}
func (WidePublicAPI) B() {}
func (WidePublicAPI) C() {}
func (WidePublicAPI) D() {}

// RaisedExported trips the total cap but raises its own exported line to 10, so
// only the total-method diagnostic fires.
//
//plimsoll:max-exported-methods=10
type RaisedExported struct{} // want `type RaisedExported has 4 methods, over the load line of 3`

func (RaisedExported) A() {}
func (RaisedExported) B() {}
func (RaisedExported) C() {}
func (RaisedExported) D() {}
