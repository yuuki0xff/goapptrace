package config

import (
	"errors"
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
	Trace map[string]*Trace
}

func NewTargets() *Targets {
	return &Targets{
		Targets: make(map[TargetName]*Target),
	}
}

func (tt *Targets) Add(t *Target) error {
	if _, exists := tt.Targets[t.Name]; exists {
		return errors.New(fmt.Sprintf(`"%s" is already exists`, t.Name))
	}
	tt.Targets[t.Name] = t
	return nil
}

func (tt *Targets) Get(name TargetName) (*Target, error) {
	t, exists := tt.Targets[name]
	if !exists {
		return nil, errors.New(fmt.Sprintf(`"%s" is not found`, name))
	}
	return t, nil
}

func (tt *Targets) Delete(name TargetName) error {
	if _, exists := tt.Targets[name]; !exists {
		return errors.New(fmt.Sprintf(`"%s" is not found`, name))
	}
	delete(tt.Targets, name)
	return nil
}

func (tt *Targets) Walk(fn func(*Target) error) error {
	for _, t := range tt.Targets {
		if err := fn(t); err != nil {
			return err
		}
	}
	return nil
}
