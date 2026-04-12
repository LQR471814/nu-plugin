package nu

import (
	"context"
	"fmt"
	"reflect"

	"github.com/vmihailenco/msgpack/v5"

	"github.com/ainvaltin/nu-plugin/internal/mpack"
	"github.com/ainvaltin/nu-plugin/syntaxshape"
	"github.com/ainvaltin/nu-plugin/types"
)

/*
Command describes an command provided by the plugin.
*/
type Command struct {
	Signature PluginSignature
	Examples  []Example

	// callback executed on command invocation
	OnRun func(context.Context, *ExecCommand) error
}

func (c Command) Validate() error {
	if err := c.Signature.Validate(); err != nil {
		return err
	}
	if c.OnRun == nil {
		return fmt.Errorf("command must have on-run handler")
	}
	return nil
}

func (c Command) encodeMsgpack(enc *msgpack.Encoder, p *Plugin) (err error) {
	if err = enc.EncodeMapLen(2); err != nil {
		return err
	}
	if err = enc.EncodeString("sig"); err != nil {
		return err
	}
	if err = c.Signature.encodeMsgpack(enc, p); err != nil {
		return err
	}
	if err = enc.EncodeString("examples"); err != nil {
		return err
	}
	return encodeExamples(enc, c.Examples, p)
}

type PluginSignature struct {
	Name string
	// This should be a single sentence as it is the part shown for example in the completion menu.
	Desc string
	// Additional documentation of the command.
	Description        string
	SearchTerms        []string
	Category           string // https://docs.rs/nu-protocol/latest/nu_protocol/enum.Category.html
	RequiredPositional []PositionalArg
	OptionalPositional []PositionalArg
	RestPositional     *PositionalArg

	// The "help" (short "h") flag will be added automatically when plugin
	// is created, do not use these names for other flags or arguments.
	Named                []Flag
	InputOutputTypes     []InOutTypes
	IsFilter             bool
	CreatesScope         bool
	AllowsUnknownArgs    bool
	AllowMissingExamples bool
}

func (sig *PluginSignature) addHelp() error {
	for _, v := range sig.Named {
		if v.Long == "help" || v.Short == 'h' {
			return fmt.Errorf("help flag is already registered")
		}
	}
	sig.Named = append(sig.Named, Flag{Long: "help", Short: 'h', Desc: "Display the help message for this command"})
	return nil
}

func (sig PluginSignature) encodeMsgpack(enc *msgpack.Encoder, p *Plugin) (err error) {
	cnt := 13 + bval(sig.RestPositional != nil)
	if err = enc.EncodeMapLen(cnt); err != nil {
		return err
	}

	if err = encodeString(enc, "name", sig.Name); err != nil {
		return err
	}
	if err = encodeString(enc, "description", sig.Desc); err != nil {
		return err
	}
	if err = encodeString(enc, "extra_description", sig.Description); err != nil {
		return err
	}
	if err = encodeString(enc, "category", sig.Category); err != nil {
		return err
	}

	if err = enc.EncodeString("search_terms"); err != nil {
		return fmt.Errorf("encoding search_terms key: %w", err)
	}
	if err = enc.EncodeArrayLen(len(sig.SearchTerms)); err != nil {
		return fmt.Errorf("encoding search_terms array length: %w", err)
	}
	for _, v := range sig.SearchTerms {
		if err = enc.EncodeString(v); err != nil {
			return err
		}
	}

	if err = enc.EncodeString("required_positional"); err != nil {
		return err
	}
	if err = encodePositionalArgs(enc, sig.RequiredPositional, p); err != nil {
		return err
	}
	if err = enc.EncodeString("optional_positional"); err != nil {
		return err
	}
	if err = encodePositionalArgs(enc, sig.OptionalPositional, p); err != nil {
		return err
	}
	if sig.RestPositional != nil {
		if err = enc.EncodeString("rest_positional"); err != nil {
			return err
		}
		if err = sig.RestPositional.encodeMsgpack(enc, p); err != nil {
			return err
		}
	}

	if err = enc.EncodeString("named"); err != nil {
		return err
	}
	if err = encodeFlags(enc, sig.Named, p); err != nil {
		return err
	}
	if err = enc.EncodeString("input_output_types"); err != nil {
		return err
	}
	if err = enc.EncodeArrayLen(len(sig.InputOutputTypes)); err != nil {
		return err
	}
	for _, v := range sig.InputOutputTypes {
		if err = v.encodeMsgpack(enc); err != nil {
			return err
		}
	}
	if err = encodeBoolean(enc, "is_filter", sig.IsFilter); err != nil {
		return err
	}
	if err = encodeBoolean(enc, "creates_scope", sig.CreatesScope); err != nil {
		return err
	}
	if err = encodeBoolean(enc, "allows_unknown_args", sig.AllowsUnknownArgs); err != nil {
		return err
	}
	if err = encodeBoolean(enc, "allow_variants_without_examples", sig.AllowMissingExamples); err != nil {
		return err
	}
	return nil
}

type InOutTypes struct {
	In  types.Type
	Out types.Type
}

type PositionalArg struct {
	Name    string
	Desc    string
	Shape   syntaxshape.SyntaxShape
	VarId   uint
	Default *Value
	// Either [StaticCompletions] or [DynamicCompletion] to provide suggestions for argument autocomplete.
	Completions Completions
}

/*
Flag is a definition of a flag (Shape is unassigned) or named argument (Shape is assigned).
*/
type Flag struct {
	Long     string // long name of the flag
	Short    rune   // optional short name of the flag
	Shape    syntaxshape.SyntaxShape
	Required bool
	Desc     string
	VarId    uint
	Default  *Value
	// Either [StaticCompletions] or [DynamicCompletion] to provide suggestions for flag autocomplete.
	Completions Completions
}

type Example struct {
	Example     string
	Description string
	Result      *Value
}

func (sig PluginSignature) Validate() error {
	if sig.Name == "" {
		return fmt.Errorf("command must have name")
	}
	if sig.Category == "" {
		return fmt.Errorf("command must have Category")
	}
	if sig.Desc == "" {
		return fmt.Errorf("command Desc must have value")
	}
	if len(sig.SearchTerms) == 0 {
		return fmt.Errorf("command Search Terms must have value")
	}
	if len(sig.InputOutputTypes) == 0 {
		return fmt.Errorf("command Input-Output types must be specified")
	}

	return nil
}

/*
Decode top-level "plugin input" message, the message must be "map".
*/
func (p *Plugin) decodeInputMsg(dec *msgpack.Decoder) (interface{}, error) {
	name, err := decodeWrapperMap(dec)
	if err != nil {
		return nil, fmt.Errorf("decode message's map: %w", err)
	}
	return p.handleMsgDecode(dec, name)
}

func (p *Plugin) handleMsgDecode(dec *msgpack.Decoder, name string) (_ any, err error) {
	switch name {
	case "Call":
		return decodeCall(dec, p)
	case "Data":
		m := data{}
		return m, m.decodeMsgpack(dec, p)
	case "Ack":
		m := ack{}
		m.ID, err = dec.DecodeInt()
		return m, err
	case "End":
		m := end{}
		m.ID, err = dec.DecodeInt()
		return m, err
	case "Drop":
		m := drop{}
		m.ID, err = dec.DecodeInt()
		return m, err
	case "EngineCallResponse":
		m := engineCallResponse{}
		return m, m.decodeMsgpack(dec, p)
	case "Hello":
		m := hello{}
		return m, dec.DecodeValue(reflect.ValueOf(&m))
	case "Signal":
		m := signal{}
		if m.Signal, err = dec.DecodeString(); m.Signal == "Interrupt" {
			return nil, ErrInterrupt
		}
		return m, err
	default:
		return nil, fmt.Errorf("unknown message %q", name)
	}
}

func encodePositionalArgs(enc *msgpack.Encoder, pa []PositionalArg, p *Plugin) error {
	if err := enc.EncodeArrayLen(len(pa)); err != nil {
		return err
	}
	for _, v := range pa {
		if err := v.encodeMsgpack(enc, p); err != nil {
			return err
		}
	}
	return nil
}

func (arg *PositionalArg) encodeMsgpack(enc *msgpack.Encoder, p *Plugin) (err error) {
	items := mpack.MapItems{
		mpack.EncoderFuncString("name", arg.Name),
		mpack.EncoderFuncString("desc", arg.Desc),
		mpack.EncoderFuncMarshal("shape", arg.Shape.EncodeMsgpack),
	}
	items.AddOptionalUInt("var_id", arg.VarId)
	items.AddOptionalEncoder(arg.Default != nil, func(enc *msgpack.Encoder) error {
		if err = enc.EncodeString("default_value"); err != nil {
			return fmt.Errorf("encoding default value key: %w", err)
		}
		if err = arg.Default.encodeMsgpack(enc, p); err != nil {
			return fmt.Errorf("encoding default value: %w", err)
		}
		return nil
	})
	if arg.Completions != nil && arg.Completions.isEncodedCompletion() {
		items = append(items, arg.Completions.encodeMsgpack)
	}

	return items.EncodeMap("PositionalArg", enc)
}

func (flag *Flag) encodeMsgpack(enc *msgpack.Encoder, p *Plugin) (err error) {
	items := mpack.MapItems{
		mpack.EncoderFuncString("long", flag.Long),
		mpack.EncoderFuncString("short", string(flag.Short)),
		mpack.EncoderFuncString("desc", flag.Desc),
		mpack.EncoderFuncBool("required", flag.Required),
	}
	items.AddOptionalUInt("var_id", flag.VarId)
	if flag.Shape != nil {
		items = append(items, mpack.EncoderFuncMarshal("arg", flag.Shape.EncodeMsgpack))
	}
	items.AddOptionalEncoder(flag.Default != nil, func(enc *msgpack.Encoder) error {
		if err = enc.EncodeString("default_value"); err != nil {
			return fmt.Errorf("encoding default value key: %w", err)
		}
		if err = flag.Default.encodeMsgpack(enc, p); err != nil {
			return fmt.Errorf("encoding default value: %w", err)
		}
		return nil
	})
	if flag.Completions != nil && flag.Completions.isEncodedCompletion() {
		items = append(items, flag.Completions.encodeMsgpack)
	}

	return items.EncodeMap("PositionalArg", enc)
}

func encodeFlags(enc *msgpack.Encoder, flags []Flag, p *Plugin) error {
	if err := enc.EncodeArrayLen(len(flags)); err != nil {
		return err
	}
	for _, v := range flags {
		if err := v.encodeMsgpack(enc, p); err != nil {
			return err
		}
	}
	return nil
}

func encodeExamples(enc *msgpack.Encoder, ex []Example, p *Plugin) error {
	if err := enc.EncodeArrayLen(len(ex)); err != nil {
		return err
	}
	for _, v := range ex {
		if err := v.encodeMsgpack(enc, p); err != nil {
			return err
		}
	}
	return nil
}

func (ex *Example) encodeMsgpack(enc *msgpack.Encoder, p *Plugin) (err error) {
	cnt := 2 + bval(ex.Result != nil)
	if err = enc.EncodeMapLen(cnt); err != nil {
		return err
	}
	if err = encodeString(enc, "description", ex.Description); err != nil {
		return err
	}
	if err = encodeString(enc, "example", ex.Example); err != nil {
		return err
	}
	if ex.Result != nil {
		if err = enc.EncodeString("result"); err != nil {
			return err
		}
		return ex.Result.encodeMsgpack(enc, p)
	}
	return nil
}

func (iot *InOutTypes) encodeMsgpack(enc *msgpack.Encoder) error {
	if err := enc.EncodeArrayLen(2); err != nil {
		return err
	}
	if err := iot.In.EncodeMsgpack(enc); err != nil {
		return err
	}
	return iot.Out.EncodeMsgpack(enc)
}
