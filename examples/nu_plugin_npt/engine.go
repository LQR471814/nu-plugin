package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
			Named: []nu.Flag{
				{Long: "named", Short: 'n', Desc: "Named arguments / flags for the engine call", Shape: syntaxshape.Record(nil)},
			},
			RestPositional: &nu.PositionalArg{Name: "arguments", Desc: "Positional arguments for the engine call", Shape: syntaxshape.Any()},
			RequiredPositional: []nu.PositionalArg{
				{
					Name:  "call",
					Desc:  "API to call",
					Shape: syntaxshape.String(),
					Completions: nu.DynamicCompletion(func() []nu.DynamicSuggestion {
						return []nu.DynamicSuggestion{
							{Value: "GetPluginConfig", Description: "Get the configuration for the plugin"},
							{Value: "GetEnvVars", Description: "Get all environment variables from the caller's scope"},
							{Value: "GetEnvVar", Description: "Get an environment variable from the caller's scope"},
							{Value: "AddEnvVar", Description: "Set an environment variable in the caller's scope"},
							{Value: "GetCurrentDir", Description: "Get the current directory path in the caller's scope"},
							{Value: "GetHelp", Description: "Get fully formatted help text for the current command"},
							{Value: "GetSpanContents", Description: "Get the contents of a Span from the engine"},
							{Value: "EvalClosure", Description: "Pass a Closure and optional arguments to the engine to be evaluated"},
							{Value: "FindDeclaration", Description: "Find the declaration ID for a command in scope"},
							{Value: "CallDecl", Description: `Call "FindDeclaration" and then call it (if declaration was found)`},
						}
					})},
			},
			AllowMissingExamples: true,
		},
		Examples: []nu.Example{
			{Description: "Get value of an env var", Example: `npt engine GetEnvVar NU_VERSION`, Result: &nu.Value{Value: "0.111.0"}},
			{Description: "Call closure with both input (bar) and positional argument (foo)", Example: `"bar" | npt engine EvalClosure { | arg | $arg + $in } "foo"`, Result: &nu.Value{Value: "foobar"}},
			{Description: `List the content of directory "/usr"`, Example: "npt engine CallDecl ls /usr"},
			{Description: "Pass named argument (switch) to command", Example: `npt engine -n {full-paths: null} CallDecl ls`},
		},
		OnRun: handleCmdEngineCalls,
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
	case "EvalClosure":
		args, err := getEvalArguments(ec)
		if err != nil {
			return err
		}
		v, err := ec.EvalClosure(ctx, ec.Positional[1], args...)
		if err != nil {
			return err
		}
		return returnValue(ctx, ec, v)
	case "FindDeclaration":
		dec, err := findDeclaration(ctx, ec)
		if err != nil {
			return err
		}
		return ec.ReturnValue(ctx, nu.ToValue(fmt.Sprintf("%#v", dec)))
	case "CallDecl":
		dec, err := findDeclaration(ctx, ec)
		if err != nil {
			return err
		}

		args, err := getEvalArguments(ec)
		if err != nil {
			return err
		}

		if np, ok := ec.FlagValue("named"); ok {
			r, ok := np.Value.(nu.Record)
			if !ok {
				return fmt.Errorf("expected Record, got %T", np)
			}
			args = append(args, nu.NamedParams(r))
		}

		v, err := dec.Call(ctx, args...)
		if err != nil {
			return err
		}
		return returnValue(ctx, ec, v)
	default:
		return nuError(fmt.Sprintf("unsupported engine call %s", ec.Positional[0].Value), "unsupported API name", ec.Positional[0].Span)
	}
}

func getEvalArguments(ec *nu.ExecCommand) ([]nu.EvalArgument, error) {
	var args []nu.EvalArgument

	switch in := ec.Input.(type) {
	case nil:
	case nu.Value:
		args = append(args, nu.InputValue(in))
	case io.Reader:
		args = append(args, nu.InputRawStream(in))
	case <-chan nu.Value:
		args = append(args, nu.InputListStream(in))
	default:
		return nil, fmt.Errorf("unsupported input type %T", in)
	}

	if len(ec.Positional) > 2 {
		args = append(args, nu.Positional(ec.Positional[2:]...))
	}
	return args, nil
}

func returnValue(ctx context.Context, ec *nu.ExecCommand, v any) error {
	switch tv := v.(type) {
	case nil:
		return nil
	case nu.Value:
		return ec.ReturnValue(ctx, tv)
	case io.Reader:
		out, err := ec.ReturnRawStream(ctx)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, tv)
		return err
	case <-chan nu.Value:
		out, err := ec.ReturnListStream(ctx)
		if err != nil {
			return fmt.Errorf("opening return list: %w", err)
		}
		defer close(out)

		for {
			select {
			case v, ok := <-tv:
				if !ok {
					return nil // input closed, all OK
				}
				select {
				case out <- v:
				case <-ctx.Done():
					return ctxErr(ctx)
				}
			case <-ctx.Done():
				return ctxErr(ctx)
			}
		}
	}
	return fmt.Errorf("unsupported value type: %T", v)
}

func findDeclaration(ctx context.Context, ec *nu.ExecCommand) (*nu.Declaration, error) {
	if len(ec.Positional) < 2 {
		return nil, fmt.Errorf("second argument is expected to be the name of the command")
	}
	name, err := argNAsType[string](ec, 1, "name of the command")
	if err != nil {
		return nil, err
	}
	return ec.FindDeclaration(ctx, name)
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
