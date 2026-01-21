package main

import (
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
)

func generateUnaryMethod(
	g *protogen.GeneratedFile,
	file *protogen.File,
	service *protogen.Service,
	method *protogen.Method,
	opts MethodCgoOptions,
) {
	serviceName := service.GoName
	methodName := method.GoName
	abiPrefix := fmt.Sprintf("Ygrpc_%s_%s", serviceName, methodName)
	reqType := g.QualifiedGoIdent(method.Input.GoIdent)
	adaptorFunc := serviceName + "_" + methodName
	adaptorCall := g.QualifiedGoIdent(file.GoImportPath.Ident(adaptorFunc))

	if shouldGenerateStandard(opts.ReqFreeMode) {
		generateUnaryBinary(g, abiPrefix, reqType, adaptorCall)
	}
	if shouldGenerateTakeReq(opts.ReqFreeMode) {
		generateUnaryBinaryTakeReq(g, abiPrefix, reqType, adaptorCall)
	}

	if shouldGenerateNative(opts.NativeMode) {
		reqFlat := isMessageFlat(method.Input)
		respFlat := isMessageFlat(method.Output)

		if reqFlat && respFlat {
			if shouldGenerateStandard(opts.ReqFreeMode) {
				generateUnaryNative(g, abiPrefix, method.Input, method.Output, adaptorCall)
			}
			if shouldGenerateTakeReq(opts.ReqFreeMode) {
				generateUnaryNativeTakeReq(g, abiPrefix, method.Input, method.Output, adaptorCall)
			}
		}
	}
}

func generateUnaryBinary(
	g *protogen.GeneratedFile,
	abiPrefix string,
	reqType string,
	adaptorCall string,
) {
	g.P("//export ", abiPrefix)
	g.P("func ", abiPrefix, "(")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen C.int,")
	g.P("    respPtr *", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    respLen *C.int,")
	g.P("    respFree *C.FreeFunc,")
	g.P(") C.int {")
	g.P("    reqBytes := C.GoBytes(reqPtr, reqLen)")
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    resp, err := ", adaptorCall, "(ctx, req)")
	g.P("    if err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()
	g.P("    respBytes, err := ", g.QualifiedGoIdent(protoPackage.Ident("Marshal")), "(resp)")
	g.P("    if err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()
	g.P("    buf := C.CBytes(respBytes)")
	g.P("    *respPtr = buf")
	g.P("    *respLen = C.int(len(respBytes))")
	g.P("    *respFree = (C.FreeFunc)(C.Ygrpc_Free)")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateUnaryBinaryTakeReq(
	g *protogen.GeneratedFile,
	abiPrefix string,
	reqType string,
	adaptorCall string,
) {
	funcName := abiPrefix + "_TakeReq"

	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen C.int,")
	g.P("    reqFree C.FreeFunc,")
	g.P("    respPtr *", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    respLen *C.int,")
	g.P("    respFree *C.FreeFunc,")
	g.P(") C.int {")
	g.P("    reqBytes := C.GoBytes(reqPtr, reqLen)")
	g.P("    if reqFree != nil {")
	g.P("        C.call_free_func(reqFree, reqPtr)")
	g.P("    }")
	g.P()
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    resp, err := ", adaptorCall, "(ctx, req)")
	g.P("    if err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()
	g.P("    respBytes, err := ", g.QualifiedGoIdent(protoPackage.Ident("Marshal")), "(resp)")
	g.P("    if err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()
	g.P("    buf := C.CBytes(respBytes)")
	g.P("    *respPtr = buf")
	g.P("    *respLen = C.int(len(respBytes))")
	g.P("    *respFree = (C.FreeFunc)(C.Ygrpc_Free)")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateUnaryNative(
	g *protogen.GeneratedFile,
	abiPrefix string,
	reqMsg *protogen.Message,
	respMsg *protogen.Message,
	adaptorCall string,
) {
	funcName := abiPrefix + "_Native"
	reqType := g.QualifiedGoIdent(reqMsg.GoIdent)

	g.P("//export ", funcName)
	g.P("func ", funcName, "(")

	generateNativeReqParams(g, reqMsg)

	generateNativeRespParams(g, respMsg)

	g.P(") C.int {")
	g.P("    req := &", reqType, "{}")

	generateNativeReqAssignments(g, reqMsg)

	g.P()
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    resp, err := ", adaptorCall, "(ctx, req)")
	g.P("    if err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()

	generateNativeRespAssignments(g, respMsg)

	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateUnaryNativeTakeReq(
	g *protogen.GeneratedFile,
	abiPrefix string,
	reqMsg *protogen.Message,
	respMsg *protogen.Message,
	adaptorCall string,
) {
	funcName := abiPrefix + "_Native_TakeReq"
	reqType := g.QualifiedGoIdent(reqMsg.GoIdent)

	g.P("//export ", funcName)
	g.P("func ", funcName, "(")

	generateNativeReqParamsTakeReq(g, reqMsg)

	generateNativeRespParams(g, respMsg)

	g.P(") C.int {")
	g.P("    req := &", reqType, "{}")

	generateNativeReqAssignmentsTakeReq(g, reqMsg)

	g.P()
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    resp, err := ", adaptorCall, "(ctx, req)")
	g.P("    if err != nil {")
	g.P("        return C.int(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P()

	generateNativeRespAssignments(g, respMsg)

	g.P("    return 0")
	g.P("}")
	g.P()
}
