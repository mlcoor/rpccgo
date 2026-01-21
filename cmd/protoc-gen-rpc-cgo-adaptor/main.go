package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"
	"strings"

	_ "embed"

	"google.golang.org/protobuf/compiler/protogen"
)

//go:embed version.txt
var version string

// ProtocolOption controls which protocol adaptor code to generate.
//
// Note: this is codegen-only; runtime protocol identifiers live in rpcruntime.Protocol.
type ProtocolOption string

const (
	ProtocolOptionConnectRPC ProtocolOption = "connectrpc"
	ProtocolOptionGrpc       ProtocolOption = "grpc"
)

// GeneratorOptions holds all options for code generation.
type GeneratorOptions struct {
	Protocols            []ProtocolOption
	ConnectPackageSuffix string
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version" || os.Args[1] == "-v") {
		fmt.Fprintln(os.Stdout, version)
		os.Exit(0)
	}

	var flags flag.FlagSet

	connectPackageSuffixFlag := flags.String(
		"connect_package_suffix",
		"",
		"connect-go package_suffix; when set, connect handler interface is assumed to be in <import-path>/<go-package-name><suffix>",
	)

	protocolFlag := flags.String(
		"protocol",
		"",
		"protocols to generate support for; use '|' to separate multiple protocols (e.g. protocol=grpc|connectrpc); default is connectrpc",
	)

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		suffix, err := parseConnectPackageSuffix(*connectPackageSuffixFlag)
		if err != nil {
			return err
		}
		protocols, err := parseProtocolToken(*protocolFlag)
		if err != nil {
			return err
		}
		genOpts := GeneratorOptions{
			Protocols:            protocols,
			ConnectPackageSuffix: suffix,
		}

		for _, f := range gen.Files {
			if !f.Generate || len(f.Services) == 0 {
				continue
			}
			generateFile(gen, f, genOpts)
		}

		return nil
	})
}

func parseProtocolToken(raw string) ([]ProtocolOption, error) {
	trimmedRaw := strings.TrimSpace(raw)
	if trimmedRaw == "" {
		return []ProtocolOption{ProtocolOptionConnectRPC}, nil
	}
	seen := make(map[ProtocolOption]bool, 2)
	var out []ProtocolOption

	if strings.Contains(trimmedRaw, ",") {
		return nil, fmt.Errorf(
			"invalid protocol value %q: use '|' to separate multiple protocols (e.g. protocol=grpc|connectrpc); commas are reserved to separate protoc plugin options",
			trimmedRaw,
		)
	}

	if strings.Contains(trimmedRaw, ";") {
		return nil, fmt.Errorf(
			"invalid protocol value %q: use '|' to separate multiple protocols (e.g. protocol=grpc|connectrpc); ';' is not supported",
			trimmedRaw,
		)
	}

	parts := strings.Split(trimmedRaw, "|")
	for _, part := range parts {
		token := strings.ToLower(strings.TrimSpace(part))
		if token == "" {
			continue
		}
		var p ProtocolOption
		switch token {
		case string(ProtocolOptionGrpc):
			p = ProtocolOptionGrpc
		case string(ProtocolOptionConnectRPC):
			p = ProtocolOptionConnectRPC
		default:
			return nil, fmt.Errorf("invalid protocol option %q (allowed: grpc, connectrpc)", token)
		}

		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}

	if len(out) == 0 {
		return []ProtocolOption{ProtocolOptionConnectRPC}, nil
	}
	return out, nil
}

func parseConnectPackageSuffix(s string) (string, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", nil
	}
	if token.IsIdentifier(trimmed) {
		return trimmed, nil
	}
	return "", fmt.Errorf("invalid connect_package_suffix %q (must be a Go identifier)", trimmed)
}

func supportsProtocol(protocols []ProtocolOption, p ProtocolOption) bool {
	for _, got := range protocols {
		if got == p {
			return true
		}
	}
	return false
}
