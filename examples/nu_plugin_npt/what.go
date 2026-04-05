package main

import (
	"context"
	"fmt"
	"io"

	"github.com/ainvaltin/nu-plugin"
	"github.com/ainvaltin/nu-plugin/syntaxshape"
	"github.com/ainvaltin/nu-plugin/types"
)

func cmdWhat() *nu.Command {
	return &nu.Command{
		Signature: nu.PluginSignature{
			Name:        "npt what",
			Desc:        "Returns information about how the plugin sees the input",
			Description: "An example how to read input and return result from a plugin.",
			Category:    "Debug",
			SearchTerms: []string{"input", "output"},
			InputOutputTypes: []nu.InOutTypes{
				{In: types.Any(), Out: types.Any()},
			},
			OptionalPositional: []nu.PositionalArg{
				{Name: "input", Desc: "input for the command", Shape: syntaxshape.Any()},
			},
			RestPositional: &nu.PositionalArg{
				Name: "rest", Desc: "Allow unknown number of optional arguments", Shape: syntaxshape.Any(),
			},
			Named: []nu.Flag{
				{Long: "items", Short: 'i', Shape: syntaxshape.Int(), Default: &nu.Value{Value: 3}, Desc: "maximum number of items to output for Lists"},
			},
			AllowMissingExamples: true,
		},
		Examples: []nu.Example{},
		OnRun:    handleCmdWhat,
	}
}

func handleCmdWhat(ctx context.Context, ec *nu.ExecCommand) error {
	return ec.ReturnValue(ctx, nu.Value{
		Value: nu.Record{
			"input":      getInputInfo(ec),
			"named":      getNamedInfo(ec),
			"positional": getPositionalInfo(ec),
		},
	})
}

func getPositionalInfo(ec *nu.ExecCommand) nu.Value {
	positional := nu.Record{}
	positional["count"] = nu.Value{Value: len(ec.Positional)}

	items := []nu.Value{}
	for _, v := range ec.Positional {
		items = append(items, nu.Value{
			Value: nu.Record{
				"type":  nu.Value{Value: fmt.Sprintf("%T", v.Value)},
				"Value": v,
				"span":  nu.ToValue(v.Span),
			}})
	}
	positional["items"] = nu.Value{Value: items}

	return nu.Value{Value: positional}
}

func getNamedInfo(ec *nu.ExecCommand) nu.Value {
	named := nu.Record{}
	named["count"] = nu.Value{Value: len(ec.Named)}

	items := []nu.Value{}
	for _, v := range ec.Named {
		items = append(items, nu.Value{
			Value: nu.Record{
				"type":  nu.Value{Value: fmt.Sprintf("%T", v.Value)},
				"Value": v,
				"span":  nu.ToValue(v.Span),
			}})
	}
	named["items"] = nu.Value{Value: items}

	return nu.Value{Value: named}
}

func getInputInfo(ec *nu.ExecCommand) nu.Value {
	input := nu.Record{}
	input["kind"] = nu.ToValue(fmt.Sprintf("%T", ec.Input))

	switch in := ec.Input.(type) {
	case nu.Value:
		input["Value"] = in
		input["Span"] = nu.ToValue(in.Span)
		input["type"] = nu.Value{Value: fmt.Sprintf("%T", in.Value)}
	case <-chan nu.Value:
		// flag how many items to read?
		total := int64(0)
		cv, _ := ec.FlagValue("items")
		cnt := cv.Value.(int64)
		items := []nu.Value{}
		for v := range in {
			items = append(items, nu.Value{
				Value: nu.Record{
					"type":  nu.Value{Value: fmt.Sprintf("%T", v.Value)},
					"value": v,
					"span":  nu.ToValue(v.Span),
				}})
			// we intentionally add at least one value into the output
			if total++; total >= cnt {
				break
			}
		}
		for range in {
			total++
		}
		input["items"] = nu.Value{Value: items}
		input["count"] = nu.Value{Value: total}
	case io.Reader:
		// try to detect is it binary or text? show fist n bytes?
		cnt, err := io.Copy(io.Discard, in)
		if err != nil {
			input["read error"] = nu.Value{Value: err}
		}
		input["bytes"] = nu.Value{Value: nu.Filesize(cnt)}
	}

	return nu.Value{Value: input}
}
