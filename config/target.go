package config

import (
	"errors"
	"fmt"
)

type TargetName string

type Targets struct {
	targets map[TargetName]*Target
}

type Target struct {
	Name  TargetName
	Files []string
	Build BuildProcess
	Trace map[string]*Trace
}

func NewTargets() *Targets {
	return &Targets{
		targets: make(map[TargetName]*Target),
	}
}

func (tt *Targets) Add(t *Target) error {
	if _, exists := tt.targets[t.Name]; exists {
		return errors.New(fmt.Sprintf(`"%s" is already exists`, t.Name))
	}
	tt.targets[t.Name] = t
	return nil
}

func (tt *Targets) Delete(name TargetName) error {
	if _, exists := tt.targets[name]; !exists {
		return errors.New(fmt.Sprintf(`"%s" is not found`, name))
	}
	delete(tt.targets, name)
	return nil
}

func (tt *Targets) Walk(fn func(*Target) error) error {
	for _, t := range tt.targets {
		if err := fn(t); err != nil {
			return err
		}
	}
	return nil
}
