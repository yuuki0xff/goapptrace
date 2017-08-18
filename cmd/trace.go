package cmd

import (
	"fmt"
	"github.com/yuuki0xff/goapptrace/config"
	"strconv"
)

func walkTrace(args CommandArgs, fn func(name config.TargetName, trace *config.Trace, created bool) error) error {
	names := args.Arguments
	if len(args.Arguments) == 0 {
		names = args.Config.Targets.Names()
	}

	for _, name := range names {
		t, err := args.Config.Targets.Get(config.TargetName(name))
		if err != nil {
			return err
		}

		if t.Trace == nil {
			t.Trace = map[string]*config.Trace{}
		}
		for _, fname := range t.Files {
			_, exists := t.Trace[fname]
			if !exists {
				t.Trace[fname] = &config.Trace{
					IsTracing: true,
				}
			}

			if err := fn(t.Name, t.Trace[fname], !exists); err != nil {
				return err
			}
		}
	}
	return nil
}

func trace_on(args CommandArgs) (int, error) {
	if err := walkTrace(args, func(name config.TargetName, trace *config.Trace, created bool) error {
		// TODO: add tracing code

		if created {
			trace.HasTracingCode = true // TODO: currently always true
			trace.IsTracing = true
		} else {
			trace.HasTracingCode = true
		}
		return nil
	}); err != nil {
		return 1, err
	}
	return 0, nil
}

func trace_off(args CommandArgs) (int, error) {
	if err := walkTrace(args, func(name config.TargetName, trace *config.Trace, created bool) error {
		// TODO: remove tracing code

		trace.HasTracingCode = false
		trace.IsTracing = false
		return nil
	}); err != nil {
		return 1, err
	}
	return 0, nil
}

func trace_status(args CommandArgs) (int, error) {
	if err := walkTrace(args, func(name config.TargetName, trace *config.Trace, created bool) error {
		fmt.Printf(
			"%s %s %s\n",
			name,
			strconv.FormatBool(trace.HasTracingCode),
			strconv.FormatBool(trace.IsTracing),
		)
		return nil
	}); err != nil {
		return 1, err
	}
	return 0, nil
}

func trace_start(args CommandArgs) (int, error) {
	if err := walkTrace(args, func(name config.TargetName, trace *config.Trace, created bool) error {
		if trace.HasTracingCode {
			// TODO: start tracing
			trace.IsTracing = true
		}
		return nil
	}); err != nil {
		return 1, err
	}
	return 0, nil
}

func trace_stop(args CommandArgs) (int, error) {
	if err := walkTrace(args, func(name config.TargetName, trace *config.Trace, created bool) error {
		// TODO: stop tracing

		trace.IsTracing = false
		return nil
	}); err != nil {
		return 1, err
	}
	return 0, nil
}
