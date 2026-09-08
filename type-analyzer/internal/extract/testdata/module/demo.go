package demo

type Reader interface{ Read() }

type Inner struct{}

type Base struct{}

func (Base) Read() {}

type PtrOnly struct{}

func (*PtrOnly) Read() {}

type Outer struct {
	Inner
	Ptr   *Inner
	Value Inner
	Items []Inner
}

func (*Outer) Run() {}
