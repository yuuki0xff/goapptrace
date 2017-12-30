package config

import (
	"fmt"
)

type TargetName string

type Targets struct {
	Targets map[TargetName]*Target
}

type Target struct {
	Name  TargetName
	Files []string
	Build BuildProcess
	Run   ExecProcess
	Trace map[string]*Trace
}

func NewTargets() *Targets {
	return &Targets{
		Targets: make(map[TargetName]*Target),
	}
}

func (tt *Targets) Add(t *Target) error {
	if _, exists := tt.Targets[t.Name]; exists {
		return fmt.Errorf(`"%s" is already exists`, t.Name)
	}
	tt.Targets[t.Name] = t
	return nil
}

func (tt *Targets) Get(name TargetName) (*Target, error) {
	t, exists := tt.Targets[name]
	if !exists {
		return nil, fmt.Errorf(`"%s" is not found`, name)
	}
	return t, nil
}

func (tt *Targets) Delete(name TargetName) error {
	if _, exists := tt.Targets[name]; !exists {
		return fmt.Errorf(`"%s" is not found`, name)
	}
	delete(tt.Targets, name)
	return nil
}

func (tt *Targets) Walk(names []string, fn func(*Target) error) error {
	if names == nil || len(names) == 0 {
		// iterate all targets
		for _, t := range tt.Targets {
			if err := fn(t); err != nil {
				return err
			}
		}
	} else {
		// iterate on names array
		for _, name := range names {
			t := tt.Targets[TargetName(name)]
			if err := fn(t); err != nil {
				return err
			}
		}
	}
	return nil
}

func (tt *Targets) Names() []string {
	names := make([]string, 0, len(tt.Targets))
	for name := range tt.Targets {
		names = append(names, string(name))
	}
	return names
}

func (t *Target) WalkTraces(files []string, fn func(fname string, trace *Trace, created bool) error) error {
	if files == nil || len(files) == 0 {
		files = t.Files
	}

	if t.Trace == nil {
		t.Trace = map[string]*Trace{}
	}

	for _, fname := range files {
		_, exists := t.Trace[fname]
		if !exists {
			t.Trace[fname] = &Trace{
				IsTracing: true,
			}
		}

		if err := fn(fname, t.Trace[fname], !exists); err != nil {
			return err
		}
	}
	return nil
}
