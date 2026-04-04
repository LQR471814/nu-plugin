package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/ainvaltin/nu-plugin"
	"github.com/ainvaltin/nu-plugin/syntaxshape"
	"github.com/ainvaltin/nu-plugin/types"
)

func cmdEngineCalls() *nu.Command {
	return &nu.Command{
		Signature: nu.PluginSignature{
			Name:        "npt engine",
			Desc:        "Calling Plugin Engine",
			Description: "An example how to call the plugin engine APIs.",
			Category:    "Debug",
			SearchTerms: []string{"engine call"},
			InputOutputTypes: []nu.InOutTypes{
				{In: types.Any(), Out: types.Any()},
			},
			RequiredPositional: []nu.PositionalArg{
				{
					Name:  "call",
					Desc:  "API to call",
					Shape: syntaxshape.String(),
					GetCompletions: func() []nu.DynamicSuggestion {
						return []nu.DynamicSuggestion{
							{Value: "GetPluginConfig", Description: "Get the configuration for the plugin"},
							{Value: "GetEnvVars", Description: "Get all environment variables from the caller's scope"},
							{Value: "GetEnvVar", Description: "Get an environment variable from the caller's scope"},
							{Value: "AddEnvVar", Description: "Set an environment variable in the caller's scope"},
							{Value: "GetCurrentDir", Description: "Get the current directory path in the caller's scope"},
							{Value: "GetHelp", Description: "Get fully formatted help text for the current command"},
							{Value: "GetSpanContents", Description: "Get the contents of a Span from the engine"},
						}
					}},
			},
			OptionalPositional: []nu.PositionalArg{
				{Name: "input1", Desc: "first input for the command", Shape: syntaxshape.Any()},
				{Name: "input2", Desc: "second input for the command", Shape: syntaxshape.Any()},
			},
			AllowMissingExamples: true,
		},
		Examples: []nu.Example{},
		OnRun:    handleCmdEngineCalls,
	}
}

func handleCmdEngineCalls(ctx context.Context, ec *nu.ExecCommand) error {
	switch ec.Positional[0].Value.(string) {
	case "GetPluginConfig":
		v, err := ec.GetPluginConfig(ctx)
		if err != nil {
			return err
		}
		if v == nil {
			return nil
		}
		return ec.ReturnValue(ctx, *v)
	case "GetEnvVars":
		v, err := ec.GetEnvVars(ctx)
		if err != nil {
			return err
		}
		return ec.ReturnValue(ctx, nu.ToValue(v))
	case "GetEnvVar":
		if len(ec.Positional) < 2 {
			return fmt.Errorf("second argument is expected to be the name of the env var")
		}
		name, err := argNAsType[string](ec, 1, "name of the env var")
		if err != nil {
			return err
		}
		v, err := ec.GetEnvVar(ctx, name)
		if err != nil {
			return err
		}
		if v == nil {
			return nil
		}
		return ec.ReturnValue(ctx, *v)
	case "AddEnvVar":
		if n := len(ec.Positional); n != 3 {
			return fmt.Errorf("expected two arguments (the name and value of the env var), got %d", n-1)
		}
		name, err := argNAsType[string](ec, 1, "name of the env var")
		if err != nil {
			return err
		}
		return ec.AddEnvVar(ctx, name, ec.Positional[2])
	case "GetSpanContents":
		if n := len(ec.Positional); n != 3 {
			return fmt.Errorf("expected two arguments (the start and end value of the span), got %d", n-1)
		}
		start, err := argNAsType[int64](ec, 1, "start of the span")
		if err != nil {
			return err
		}
		end, err := argNAsType[int64](ec, 2, "end of the span")
		if err != nil {
			return err
		}
		v, err := ec.GetSpanContents(ctx, nu.Span{Start: int(start), End: int(end)})
		if err != nil {
			return err
		}
		return ec.ReturnValue(ctx, nu.Value{Value: v})
	case "GetCurrentDir":
		v, err := ec.GetCurrentDir(ctx)
		if err != nil {
			return err
		}
		return ec.ReturnValue(ctx, nu.Value{Value: v})
	case "GetHelp":
		v, err := ec.GetHelp(ctx)
		if err != nil {
			return err
		}
		return ec.ReturnValue(ctx, nu.Value{Value: v})
	default:
		return nuError(fmt.Sprintf("unsupported engine call %s", ec.Positional[0].Value), "unsupported API name", ec.Positional[0].Span)
	}
}

func argNAsType[T any](ec *nu.ExecCommand, n int, varName string) (T, error) {
	par := ec.Positional[n]
	v, ok := par.Value.(T)
	if !ok {
		return v, (&nu.Error{Err: fmt.Errorf("Argument %d (%s) is of unexpected type", n, varName)}).AddLabel(fmt.Sprintf("expected %s, got %T", reflect.TypeFor[T]().Name(), par.Value), par.Span)
	}
	return v, nil
}

func nuError(msg, label string, span nu.Span) error {
	return (&nu.Error{Err: errors.New(msg)}).AddLabel(label, span)
}
