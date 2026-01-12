package main

import (
	"flag"
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// FrameworkOption controls which framework adaptor code to generate.
type FrameworkOption int

const (
	// FrameworkConnectRPC generates adaptor code only for connectrpc.
	FrameworkConnectRPC FrameworkOption = iota
	// FrameworkGrpc generates adaptor code only for grpc.
	FrameworkGrpc
)

// GeneratorOptions holds all options for code generation.
type GeneratorOptions struct {
	Framework FrameworkOption
}

func main() {
	var flags flag.FlagSet
	frameworkFlag := flags.String("framework", "connectrpc", "Framework to generate: connectrpc or grpc")
	// Note: Connect framework only supports Simple API mode.
	opts := protogen.Options{ParamFunc: flags.Set}

	opts.Run(func(gen *protogen.Plugin) error {
		genOpts := GeneratorOptions{
			Framework: parseFrameworkOption(*frameworkFlag),
		}

		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			if len(f.Services) == 0 {
				continue
			}
			generateFile(gen, f, genOpts)
		}

		return nil
	})
}

func parseFrameworkOption(s string) FrameworkOption {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "grpc":
		return FrameworkGrpc
	default:
		return FrameworkConnectRPC
	}
}
