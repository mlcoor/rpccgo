package main

import (
	_ "embed"
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"

	cgopb "github.com/ygrpc/rpccgo/proto/ygrpc/cgo"
)

//go:embed version.txt
var version string

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version" || os.Args[1] == "-v") {
		fmt.Fprintln(os.Stdout, version)
		os.Exit(0)
	}

	var flags flag.FlagSet

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		hasServices := false
		for _, f := range gen.Files {
			if f.Generate && len(f.Services) > 0 {
				hasServices = true
				break
			}
		}

		if !hasServices {
			return nil
		}

		generateCgoCommonHeader(gen)
		generateMainFile(gen)

		for _, f := range gen.Files {
			if !f.Generate || len(f.Services) == 0 {
				continue
			}
			generateCgoFile(gen, f)
		}

		return nil
	})
}

type CgoOptions struct {
	ReqFreeDefault cgopb.RequestFreeMode
	NativeDefault  cgopb.NativeMode
}

func getFileOptions(file *protogen.File) CgoOptions {
	opts := CgoOptions{
		ReqFreeDefault: cgopb.RequestFreeMode_REQ_FREE_NONE,
		NativeDefault:  cgopb.NativeMode_NATIVE_DISABLE,
	}

	if file.Desc.Options() == nil {
		return opts
	}

	if proto.HasExtension(file.Desc.Options(), cgopb.E_YgrpcCgoReqFreeDefault) {
		v := proto.GetExtension(file.Desc.Options(), cgopb.E_YgrpcCgoReqFreeDefault)
		if mode, ok := v.(cgopb.RequestFreeMode); ok {
			opts.ReqFreeDefault = mode
		}
	}

	if proto.HasExtension(file.Desc.Options(), cgopb.E_YgrpcCgoNativeDefault) {
		v := proto.GetExtension(file.Desc.Options(), cgopb.E_YgrpcCgoNativeDefault)
		if mode, ok := v.(cgopb.NativeMode); ok {
			opts.NativeDefault = mode
		}
	}

	return opts
}

type MethodCgoOptions struct {
	ReqFreeMode cgopb.RequestFreeMode
	NativeMode  cgopb.NativeMode
}

func getMethodOptions(method *protogen.Method, fileOpts CgoOptions) MethodCgoOptions {
	opts := MethodCgoOptions{
		ReqFreeMode: fileOpts.ReqFreeDefault,
		NativeMode:  fileOpts.NativeDefault,
	}

	if method.Desc.Options() == nil {
		return opts
	}

	if proto.HasExtension(method.Desc.Options(), cgopb.E_YgrpcCgoReqFreeMethod) {
		v := proto.GetExtension(method.Desc.Options(), cgopb.E_YgrpcCgoReqFreeMethod)
		if mode, ok := v.(cgopb.RequestFreeMode); ok {
			opts.ReqFreeMode = mode
		}
	}

	if proto.HasExtension(method.Desc.Options(), cgopb.E_YgrpcCgoNative) {
		v := proto.GetExtension(method.Desc.Options(), cgopb.E_YgrpcCgoNative)
		if mode, ok := v.(cgopb.NativeMode); ok {
			opts.NativeMode = mode
		}
	}

	return opts
}

func shouldGenerateStandard(mode cgopb.RequestFreeMode) bool {
	switch mode {
	case cgopb.RequestFreeMode_REQ_FREE_NONE, cgopb.RequestFreeMode_REQ_FREE_BOTH:
		return true
	case cgopb.RequestFreeMode_REQ_FREE_TAKE_REQ:
		return false
	default:
		// Forward-compatible default.
		return true
	}
}

func shouldGenerateTakeReq(mode cgopb.RequestFreeMode) bool {
	switch mode {
	case cgopb.RequestFreeMode_REQ_FREE_TAKE_REQ, cgopb.RequestFreeMode_REQ_FREE_BOTH:
		return true
	case cgopb.RequestFreeMode_REQ_FREE_NONE:
		return false
	default:
		return false
	}
}

func shouldGenerateNative(mode cgopb.NativeMode) bool {
	return mode == cgopb.NativeMode_NATIVE_ENABLE
}
