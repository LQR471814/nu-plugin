package main

import (
	"context"

	"github.com/ainvaltin/nu-plugin"
	"github.com/ainvaltin/nu-plugin/syntaxshape"
	"github.com/ainvaltin/nu-plugin/types"
)

func cmdCompletion() *nu.Command {
	return &nu.Command{
		Signature: nu.PluginSignature{
			Name:        "npt ac",
			Desc:        "Flag & Parameter autocompletion demo",
			Description: "How to implement autocompletion for flags and positional arguments.",
			Category:    "Debug",
			SearchTerms: []string{"autocomplete"},
			InputOutputTypes: []nu.InOutTypes{
				{In: types.Any(), Out: types.Any()},
			},
			Named: []nu.Flag{{
				Long:        "none",
				Short:       'n',
				Shape:       syntaxshape.String(),
				Desc:        "no custom completion",
				Completions: nil,
			}, {
				Long:        "empty",
				Short:       'e',
				Shape:       syntaxshape.Int(),
				Desc:        "dynamic completion callback but no items returned",
				Completions: nu.DynamicCompletion(func() []nu.DynamicSuggestion { return []nu.DynamicSuggestion{} }),
			}, {
				Long:  "some",
				Short: 's',
				Shape: syntaxshape.Any(),
				Desc:  "dynamic completion callback returns items",
				Completions: nu.DynamicCompletion(func() []nu.DynamicSuggestion {
					return []nu.DynamicSuggestion{
						{Value: "1", Display: "first"},
						{Value: "2", Display: "second"},
						{Value: "3", Display: "third"},
					}
				}),
			}},
			RequiredPositional: []nu.PositionalArg{
				{Name: "posarg", Desc: "positional arg", Shape: syntaxshape.Any(), Completions: completePosArg()},
			},
			OptionalPositional: []nu.PositionalArg{
				{Name: "opt1", Shape: syntaxshape.Any(), Completions: nu.StaticCompletions{"str", "[1 2]", `{name: value}`, "123"}},
			},
			AllowMissingExamples: true,
		},
		Examples: []nu.Example{},
		OnRun:    handleAutocompleteCmd,
	}
}

func completePosArg() nu.DynamicCompletion {
	return func() []nu.DynamicSuggestion {
		return []nu.DynamicSuggestion{
			{Value: "42", Description: "integer"},
			{Value: "some text", Description: "string"},
			{Value: "[1, 2]", Description: "list"},
			{Value: "{field: 88}", Description: "record"},
			{Value: "1+2", Description: "expression"},
		}
	}
}

func handleAutocompleteCmd(ctx context.Context, call *nu.ExecCommand) error {
	return nil
}
