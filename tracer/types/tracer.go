package types

type Tracer struct {
	ID     int
	Target TraceTarget
}

func (t Tracer) Copy(to *Tracer) {
	to.ID = t.ID
	t.Target.Copy(&to.Target)
}

type TraceTarget struct {
	Funcs []string
}

func (t TraceTarget) Copy(to *TraceTarget) {
	funcs := make([]string, len(t.Funcs))
	copy(funcs, t.Funcs)
	to.Funcs = funcs
}
