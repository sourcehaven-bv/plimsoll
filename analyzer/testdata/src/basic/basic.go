package basic

// Small is well under the caps — no diagnostic.
type Small struct {
	A int
	B int
}

func (Small) M1() {}
func (Small) M2() {}

// TooManyMethods has 4 methods; the test runs with maxMethods=3.
type TooManyMethods struct{} // want `type TooManyMethods has 4 methods, over the load line of 3`

func (TooManyMethods) A() {}
func (TooManyMethods) B() {}
func (TooManyMethods) C() {}
func (TooManyMethods) D() {}

// PointerRecv mixes pointer and value receivers; both count. 4 total.
type PointerRecv struct{} // want `type PointerRecv has 4 methods, over the load line of 3`

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

// Raised exceeds the default method cap but raises its own line to 10.
//
//plimsoll:max-methods=10
type Raised struct{}

func (Raised) A() {}
func (Raised) B() {}
func (Raised) C() {}
func (Raised) D() {}
func (Raised) E() {}
