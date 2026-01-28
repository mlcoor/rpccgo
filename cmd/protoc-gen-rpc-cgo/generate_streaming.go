package main

import (
	"fmt"
	"sort"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func sortedFieldsByNumber(msg *protogen.Message) []*protogen.Field {
	fields := append([]*protogen.Field(nil), msg.Fields...)
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Desc.Number() < fields[j].Desc.Number()
	})
	return fields
}

func nativeOnReadWrapperName(serviceName, methodName string) string {
	// Must match the C helper emitted by generateCgoFile.
	return fmt.Sprintf("call_on_read_native_%s_%s", serviceName, methodName)
}

func generateClientStreamingMethod(
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
	respType := g.QualifiedGoIdent(method.Output.GoIdent)
	adaptorStart := g.QualifiedGoIdent(file.GoImportPath.Ident(serviceName + "_" + methodName + "Start"))
	adaptorSend := g.QualifiedGoIdent(file.GoImportPath.Ident(serviceName + "_" + methodName + "Send"))
	adaptorFinish := g.QualifiedGoIdent(file.GoImportPath.Ident(serviceName + "_" + methodName + "Finish"))

	generateClientStreamStart(g, abiPrefix+"Start", adaptorStart)
	if shouldGenerateStandard(opts.ReqFreeMode) {
		generateClientStreamSendBinary(g, abiPrefix+"Send", reqType, adaptorSend)
	}
	if shouldGenerateTakeReq(opts.ReqFreeMode) {
		generateClientStreamSendBinaryTakeReq(g, abiPrefix+"Send_TakeReq", reqType, adaptorSend)
	}
	generateClientStreamFinishBinary(g, abiPrefix+"Finish", adaptorFinish)

	if shouldGenerateNative(opts.NativeMode) {
		reqFlat := isMessageFlat(method.Input)
		respFlat := isMessageFlat(method.Output)
		if reqFlat && respFlat {
			generateClientStreamStart(g, abiPrefix+"Start_Native", adaptorStart)
			if shouldGenerateStandard(opts.ReqFreeMode) {
				generateClientStreamSendNative(g, abiPrefix+"Send_Native", reqType, method.Input, adaptorSend)
			}
			if shouldGenerateTakeReq(opts.ReqFreeMode) {
				generateClientStreamSendNativeTakeReq(
					g,
					abiPrefix+"Send_Native_TakeReq",
					reqType,
					method.Input,
					adaptorSend,
				)
			}
			generateClientStreamFinishNative(g, abiPrefix+"Finish_Native", respType, method.Output, adaptorFinish)
		}
	}
}

func generateClientStreamStart(g *protogen.GeneratedFile, funcName string, adaptorStart string) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(outHandle *uint64) uint64 {")
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    handle, err := ", adaptorStart, "(ctx)")
	g.P("    if err != nil {")
	g.P("        *outHandle = 0")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    *outHandle = uint64(handle)")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateClientStreamSendBinary(g *protogen.GeneratedFile, funcName string, reqType string, adaptorSend string) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle C.uint64_t,")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen C.int,")
	g.P(") C.uint64_t {")
	g.P("    reqBytes := C.GoBytes(reqPtr, reqLen)")
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateClientStreamSendBinaryTakeReq(
	g *protogen.GeneratedFile,
	funcName string,
	reqType string,
	adaptorSend string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle C.uint64_t,")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen C.int,")
	g.P("    reqFree C.FreeFunc,")
	g.P(") C.uint64_t {")
	g.P("    reqBytes := C.GoBytes(reqPtr, reqLen)")
	g.P("    if reqFree != nil {")
	g.P("        C.call_free_func(reqFree, reqPtr)")
	g.P("    }")
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateClientStreamFinishBinary(
	g *protogen.GeneratedFile,
	funcName string,
	adaptorFinish string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle uint64,")
	g.P("    respPtr *", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    respLen *int,")
	g.P("    respFree *", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P(") uint64 {")
	g.P("    resp, err := ", adaptorFinish, "(uint64(streamHandle))")
	g.P("    if err != nil {")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    if resp == nil {")
	g.P("        *respPtr = nil")
	g.P("        *respLen = 0")
	g.P("        *respFree = nil")
	g.P("        return 0")
	g.P("    }")
	g.P("    respBytes, err := ", g.QualifiedGoIdent(protoPackage.Ident("Marshal")), "(resp)")
	g.P("    if err != nil {")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    buf := C.CBytes(respBytes)")
	g.P("    *respPtr = buf")
	g.P("    *respLen = len(respBytes)")
	g.P("    *respFree = (unsafe.Pointer)(C.Ygrpc_Free)")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateClientStreamSendNative(
	g *protogen.GeneratedFile,
	funcName string,
	reqType string,
	reqMsg *protogen.Message,
	adaptorSend string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle C.uint64_t,")
	generateNativeReqParams(g, reqMsg)
	g.P(") C.uint64_t {")
	g.P("    req := &", reqType, "{}")
	generateNativeReqAssignments(g, reqMsg)
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateClientStreamSendNativeTakeReq(
	g *protogen.GeneratedFile,
	funcName string,
	reqType string,
	reqMsg *protogen.Message,
	adaptorSend string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle C.uint64_t,")
	generateNativeReqParamsTakeReq(g, reqMsg)
	g.P(") C.uint64_t {")
	g.P("    req := &", reqType, "{}")
	generateNativeReqAssignmentsTakeReq(g, reqMsg)
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateClientStreamFinishNative(
	g *protogen.GeneratedFile,
	funcName string,
	respType string,
	respMsg *protogen.Message,
	adaptorFinish string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle C.uint64_t,")
	generateNativeRespParams(g, respMsg)
	g.P(") C.uint64_t {")
	g.P("    resp, err := ", adaptorFinish, "(uint64(streamHandle))")
	g.P("    if err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    if resp == nil {")
	g.P("        resp = &", respType, "{}")
	g.P("    }")
	generateNativeRespAssignments(g, respMsg)
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateServerStreamingMethod(
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
	respType := g.QualifiedGoIdent(method.Output.GoIdent)
	adaptorCall := g.QualifiedGoIdent(file.GoImportPath.Ident(serviceName + "_" + methodName))

	if shouldGenerateStandard(opts.ReqFreeMode) {
		generateServerStreamBinary(g, abiPrefix, reqType, respType, adaptorCall)
	}
	if shouldGenerateTakeReq(opts.ReqFreeMode) {
		generateServerStreamBinaryTakeReq(g, abiPrefix+"_TakeReq", reqType, respType, adaptorCall)
	}

	if shouldGenerateNative(opts.NativeMode) {
		reqFlat := isMessageFlat(method.Input)
		respFlat := isMessageFlat(method.Output)
		if reqFlat && respFlat {
			if shouldGenerateStandard(opts.ReqFreeMode) {
				generateServerStreamNative(
					g,
					serviceName,
					methodName,
					abiPrefix+"_Native",
					reqType,
					method.Input,
					respType,
					method.Output,
					adaptorCall,
				)
			}
			if shouldGenerateTakeReq(opts.ReqFreeMode) {
				generateServerStreamNativeTakeReq(
					g,
					serviceName,
					methodName,
					abiPrefix+"_Native_TakeReq",
					reqType,
					method.Input,
					respType,
					method.Output,
					adaptorCall,
				)
			}
		}
	}
}

func generateServerStreamBinary(
	g *protogen.GeneratedFile,
	funcName string,
	reqType string,
	respType string,
	adaptorCall string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen int,")
	g.P("    onReadBytes unsafe.Pointer,")
	g.P("    onDone unsafe.Pointer,")
	g.P("    callID uint64,")
	g.P(") uint64 {")
	g.P("    reqBytes := unsafe.Slice((*byte)(reqPtr), reqLen)")
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    var doneErrId ", g.QualifiedGoIdent(syncAtomicPkg.Ident("Uint64")))
	g.P("    onRead := func(resp *", respType, ") bool {")
	g.P("        respBytes, err := ", g.QualifiedGoIdent(protoPackage.Ident("Marshal")), "(resp)")
	g.P("        if err != nil {")
	g.P("            return false")
	g.P("        }")
	g.P("        respCopy := C.CBytes(respBytes)")
	g.P(
		"        C.call_on_read_bytes(onReadBytes, C.uint64_t(callID), respCopy, C.int(len(respBytes)), (C.FreeFunc)(C.Ygrpc_Free))",
	)
	g.P("        return true")
	g.P("    }")
	g.P("    onDoneFunc := func(err error) {")
	g.P("        if err != nil {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        } else {")
	g.P("            doneErrId.Store(0)")
	g.P("        }")
	g.P("        C.call_on_done(onDone, C.uint64_t(callID), C.uint64_t(doneErrId.Load()))")
	g.P("    }")
	g.P("    err := ", adaptorCall, "(ctx, req, onRead, onDoneFunc)")
	g.P("    if err != nil {")
	g.P("        if doneErrId.Load() == 0 {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        }")
	g.P("        return uint64(doneErrId.Load())")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateServerStreamBinaryTakeReq(
	g *protogen.GeneratedFile,
	funcName string,
	reqType string,
	respType string,
	adaptorCall string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen int,")
	g.P("    reqFree unsafe.Pointer,")
	g.P("    onReadBytes unsafe.Pointer,")
	g.P("    onDone unsafe.Pointer,")
	g.P("    callID uint64,")
	g.P(") uint64 {")
	g.P("    reqBytes := unsafe.Slice((*byte)(reqPtr), reqLen)")
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        if reqFree != nil {")
	g.P("            C.call_free_func((C.FreeFunc)(reqFree), reqPtr)")
	g.P("        }")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    if reqFree != nil {")
	g.P("        C.call_free_func((C.FreeFunc)(reqFree), reqPtr)")
	g.P("    }")
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    var doneErrId ", g.QualifiedGoIdent(syncAtomicPkg.Ident("Uint64")))
	g.P("    onRead := func(resp *", respType, ") bool {")
	g.P("        respBytes, err := ", g.QualifiedGoIdent(protoPackage.Ident("Marshal")), "(resp)")
	g.P("        if err != nil {")
	g.P("            return false")
	g.P("        }")
	g.P("        respCopy := C.CBytes(respBytes)")
	g.P(
		"        C.call_on_read_bytes(onReadBytes, C.uint64_t(callID), respCopy, C.int(len(respBytes)), (C.FreeFunc)(C.Ygrpc_Free))",
	)
	g.P("        return true")
	g.P("    }")
	g.P("    onDoneFunc := func(err error) {")
	g.P("        if err != nil {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        } else {")
	g.P("            doneErrId.Store(0)")
	g.P("        }")
	g.P("        C.call_on_done(onDone, C.uint64_t(callID), C.uint64_t(doneErrId.Load()))")
	g.P("    }")
	g.P("    err := ", adaptorCall, "(ctx, req, onRead, onDoneFunc)")
	g.P("    if err != nil {")
	g.P("        if doneErrId.Load() == 0 {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        }")
	g.P("        return uint64(doneErrId.Load())")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateServerStreamNative(
	g *protogen.GeneratedFile,
	serviceName, methodName, funcName string,
	reqType string,
	reqMsg *protogen.Message,
	respType string,
	respMsg *protogen.Message,
	adaptorCall string,
) {
	wrapper := nativeOnReadWrapperName(serviceName, methodName)

	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	generateNativeReqParams(g, reqMsg)
	g.P("    onReadNative unsafe.Pointer,")
	g.P("    onDone unsafe.Pointer,")
	g.P("    callID C.uint64_t,")
	g.P(") C.uint64_t {")
	g.P("    req := &", reqType, "{}")
	generateNativeReqAssignments(g, reqMsg)

	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    var doneErrId ", g.QualifiedGoIdent(syncAtomicPkg.Ident("Uint64")))
	g.P("    onRead := func(resp *", respType, ") bool {")
	g.P("        ")
	fields := sortedFieldsByNumber(respMsg)
	for _, field := range fields {
		param := nativeParamName(field, "")
		goField := field.GoName
		switch field.Desc.Kind() {
		case protoreflect.BoolKind:
			g.P("        var ", param, " C.int8_t")
			g.P("        if resp.", goField, " { ", param, " = 1 } else { ", param, " = 0 }")
		case protoreflect.StringKind:
			g.P("        var ", param, "_ptr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")))
			g.P("        var ", param, "_len C.int")
			g.P("        var ", param, "_free C.FreeFunc")
			g.P("        if len(resp.", goField, ") > 0 {")
			g.P("            ", param, "_ptr = C.CBytes([]byte(resp.", goField, "))")
			g.P("            ", param, "_len = C.int(len(resp.", goField, "))")
			g.P("            ", param, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("        }")
		case protoreflect.BytesKind:
			g.P("        var ", param, "_ptr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")))
			g.P("        var ", param, "_len C.int")
			g.P("        var ", param, "_free C.FreeFunc")
			g.P("        if len(resp.", goField, ") > 0 {")
			g.P("            ", param, "_ptr = C.CBytes(resp.", goField, ")")
			g.P("            ", param, "_len = C.int(len(resp.", goField, "))")
			g.P("            ", param, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("        }")
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			g.P("        ", param, " := C.int32_t(resp.", goField, ")")
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			g.P("        ", param, " := C.int64_t(resp.", goField, ")")
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			g.P("        ", param, " := C.uint32_t(resp.", goField, ")")
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			g.P("        ", param, " := C.uint64_t(resp.", goField, ")")
		case protoreflect.FloatKind:
			g.P("        ", param, " := C.float(resp.", goField, ")")
		case protoreflect.DoubleKind:
			g.P("        ", param, " := C.double(resp.", goField, ")")
		default:
			// Should not happen when flat-message eligibility is enforced.
			g.P("        ", param, " := C.int64_t(0)")
		}
	}
	// Invoke callback.
	g.P("        C.", wrapper, "(onReadNative, callID,")
	for _, field := range fields {
		param := nativeParamName(field, "")
		if isBytesOrStringField(field) {
			g.P("            ", param, "_ptr, ", param, "_len, ", param, "_free,")
		} else {
			g.P("            ", param, ",")
		}
	}
	g.P("        )")
	g.P("        return true")
	g.P("    }")
	g.P("    onDoneFunc := func(err error) {")
	g.P("        if err != nil {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        } else {")
	g.P("            doneErrId.Store(0)")
	g.P("        }")
	g.P("        C.call_on_done(onDone, callID, C.uint64_t(doneErrId.Load()))")
	g.P("    }")
	g.P("    err := ", adaptorCall, "(ctx, req, onRead, onDoneFunc)")
	g.P("    if err != nil {")
	g.P("        if doneErrId.Load() == 0 {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        }")
	g.P("        return C.uint64_t(doneErrId.Load())")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateServerStreamNativeTakeReq(
	g *protogen.GeneratedFile,
	serviceName, methodName, funcName string,
	reqType string,
	reqMsg *protogen.Message,
	respType string,
	respMsg *protogen.Message,
	adaptorCall string,
) {
	wrapper := nativeOnReadWrapperName(serviceName, methodName)

	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	generateNativeReqParamsTakeReq(g, reqMsg)
	g.P("    onReadNative unsafe.Pointer,")
	g.P("    onDone unsafe.Pointer,")
	g.P("    callID C.uint64_t,")
	g.P(") C.uint64_t {")
	g.P("    req := &", reqType, "{}")
	generateNativeReqAssignmentsTakeReq(g, reqMsg)

	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    var doneErrId ", g.QualifiedGoIdent(syncAtomicPkg.Ident("Uint64")))
	g.P("    onRead := func(resp *", respType, ") bool {")
	fields := sortedFieldsByNumber(respMsg)
	for _, field := range fields {
		param := nativeParamName(field, "")
		goField := field.GoName
		switch field.Desc.Kind() {
		case protoreflect.BoolKind:
			g.P("        var ", param, " C.int8_t")
			g.P("        if resp.", goField, " { ", param, " = 1 } else { ", param, " = 0 }")
		case protoreflect.StringKind:
			g.P("        var ", param, "_ptr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")))
			g.P("        var ", param, "_len C.int")
			g.P("        var ", param, "_free C.FreeFunc")
			g.P("        if len(resp.", goField, ") > 0 {")
			g.P("            ", param, "_ptr = C.CBytes([]byte(resp.", goField, "))")
			g.P("            ", param, "_len = C.int(len(resp.", goField, "))")
			g.P("            ", param, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("        }")
		case protoreflect.BytesKind:
			g.P("        var ", param, "_ptr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")))
			g.P("        var ", param, "_len C.int")
			g.P("        var ", param, "_free C.FreeFunc")
			g.P("        if len(resp.", goField, ") > 0 {")
			g.P("            ", param, "_ptr = C.CBytes(resp.", goField, ")")
			g.P("            ", param, "_len = C.int(len(resp.", goField, "))")
			g.P("            ", param, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("        }")
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			g.P("        ", param, " := C.int32_t(resp.", goField, ")")
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			g.P("        ", param, " := C.int64_t(resp.", goField, ")")
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			g.P("        ", param, " := C.uint32_t(resp.", goField, ")")
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			g.P("        ", param, " := C.uint64_t(resp.", goField, ")")
		case protoreflect.FloatKind:
			g.P("        ", param, " := C.float(resp.", goField, ")")
		case protoreflect.DoubleKind:
			g.P("        ", param, " := C.double(resp.", goField, ")")
		default:
			g.P("        ", param, " := C.int64_t(0)")
		}
	}
	g.P("        C.", wrapper, "(onReadNative, callID,")
	for _, field := range fields {
		param := nativeParamName(field, "")
		if isBytesOrStringField(field) {
			g.P("            ", param, "_ptr, ", param, "_len, ", param, "_free,")
		} else {
			g.P("            ", param, ",")
		}
	}
	g.P("        )")
	g.P("        return true")
	g.P("    }")
	g.P("    onDoneFunc := func(err error) {")
	g.P("        if err != nil {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        } else {")
	g.P("            doneErrId.Store(0)")
	g.P("        }")
	g.P("        C.call_on_done(onDone, callID, C.uint64_t(doneErrId.Load()))")
	g.P("    }")
	g.P("    err := ", adaptorCall, "(ctx, req, onRead, onDoneFunc)")
	g.P("    if err != nil {")
	g.P("        if doneErrId.Load() == 0 {")
	g.P("            doneErrId.Store(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("        }")
	g.P("        return C.uint64_t(doneErrId.Load())")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateBidiStreamingMethod(
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
	respType := g.QualifiedGoIdent(method.Output.GoIdent)
	adaptorStart := g.QualifiedGoIdent(file.GoImportPath.Ident(serviceName + "_" + methodName + "Start"))
	adaptorSend := g.QualifiedGoIdent(file.GoImportPath.Ident(serviceName + "_" + methodName + "Send"))
	adaptorCloseSend := g.QualifiedGoIdent(file.GoImportPath.Ident(serviceName + "_" + methodName + "CloseSend"))

	generateBidiStartBinary(g, abiPrefix+"Start", respType, adaptorStart)
	if shouldGenerateStandard(opts.ReqFreeMode) {
		generateBidiSendBinary(g, abiPrefix+"Send", reqType, adaptorSend)
	}
	if shouldGenerateTakeReq(opts.ReqFreeMode) {
		generateBidiSendBinaryTakeReq(g, abiPrefix+"Send_TakeReq", reqType, adaptorSend)
	}
	generateBidiCloseSend(g, abiPrefix+"CloseSend", adaptorCloseSend)

	if shouldGenerateNative(opts.NativeMode) {
		reqFlat := isMessageFlat(method.Input)
		respFlat := isMessageFlat(method.Output)
		if reqFlat && respFlat {
			generateBidiStartNative(g, serviceName, methodName, abiPrefix+"Start_Native", method.Output, adaptorStart)
			if shouldGenerateStandard(opts.ReqFreeMode) {
				generateBidiSendNative(g, abiPrefix+"Send_Native", reqType, method.Input, adaptorSend)
			}
			if shouldGenerateTakeReq(opts.ReqFreeMode) {
				generateBidiSendNativeTakeReq(g, abiPrefix+"Send_Native_TakeReq", reqType, method.Input, adaptorSend)
			}
			generateBidiCloseSend(g, abiPrefix+"CloseSend_Native", adaptorCloseSend)
		}
	}
}

func generateBidiStartBinary(g *protogen.GeneratedFile, funcName string, respType string, adaptorStart string) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    onReadBytes unsafe.Pointer,")
	g.P("    onDone unsafe.Pointer,")
	g.P("    outHandle *uint64,")
	g.P(") uint64 {")
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    handleReady := make(chan struct{})")
	g.P("    var streamHandle uint64")
	g.P("    onRead := func(resp *", respType, ") bool {")
	g.P("        <-handleReady")
	g.P("        respBytes, err := ", g.QualifiedGoIdent(protoPackage.Ident("Marshal")), "(resp)")
	g.P("        if err != nil {")
	g.P("            return false")
	g.P("        }")
	g.P("        respCopy := C.CBytes(respBytes)")
	g.P(
		"        C.call_on_read_bytes(onReadBytes, C.uint64_t(streamHandle), respCopy, C.int(len(respBytes)), (C.FreeFunc)(C.Ygrpc_Free))",
	)
	g.P("        return true")
	g.P("    }")
	g.P("    onDoneFunc := func(err error) {")
	g.P("        <-handleReady")
	g.P("        errId := uint64(0)")
	g.P("        if err != nil {")
	g.P("            errId = ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err)")
	g.P("        }")
	g.P("        C.call_on_done(onDone, C.uint64_t(streamHandle), C.uint64_t(errId))")
	g.P("    }")
	g.P("    handle, err := ", adaptorStart, "(ctx, onRead, onDoneFunc)")
	g.P("    if err != nil {")
	g.P("        *outHandle = 0")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    streamHandle = handle")
	g.P("    close(handleReady)")
	g.P("    *outHandle = uint64(handle)")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateBidiSendBinary(g *protogen.GeneratedFile, funcName string, reqType string, adaptorSend string) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle uint64,")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen int,")
	g.P(") uint64 {")
	g.P("    reqBytes := unsafe.Slice((*byte)(reqPtr), reqLen)")
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateBidiSendBinaryTakeReq(g *protogen.GeneratedFile, funcName string, reqType string, adaptorSend string) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle uint64,")
	g.P("    reqPtr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")), ",")
	g.P("    reqLen int,")
	g.P("    reqFree unsafe.Pointer,")
	g.P(") uint64 {")
	g.P("    reqBytes := unsafe.Slice((*byte)(reqPtr), reqLen)")
	g.P("    req := &", reqType, "{}")
	g.P("    if err := ", g.QualifiedGoIdent(protoPackage.Ident("Unmarshal")), "(reqBytes, req); err != nil {")
	g.P("        if reqFree != nil {")
	g.P("            C.call_free_func((C.FreeFunc)(reqFree), reqPtr)")
	g.P("        }")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    if reqFree != nil {")
	g.P("        C.call_free_func((C.FreeFunc)(reqFree), reqPtr)")
	g.P("    }")
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateBidiCloseSend(g *protogen.GeneratedFile, funcName string, adaptorCloseSend string) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(streamHandle uint64) uint64 {")
	g.P("    if err := ", adaptorCloseSend, "(uint64(streamHandle)); err != nil {")
	g.P("        return uint64(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateBidiStartNative(
	g *protogen.GeneratedFile,
	serviceName, methodName, funcName string,
	respMsg *protogen.Message,
	adaptorStart string,
) {
	wrapper := nativeOnReadWrapperName(serviceName, methodName)
	respType := g.QualifiedGoIdent(respMsg.GoIdent)

	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    onReadNative unsafe.Pointer,")
	g.P("    onDone unsafe.Pointer,")
	g.P("    outHandle *C.uint64_t,")
	g.P(") C.uint64_t {")
	g.P("    ctx := ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("BackgroundContext")), "()")
	g.P("    handleReady := make(chan struct{})")
	g.P("    var streamHandle uint64")
	g.P("    onRead := func(resp *", respType, ") bool {")
	g.P("        <-handleReady")
	fields := sortedFieldsByNumber(respMsg)
	for _, field := range fields {
		param := nativeParamName(field, "")
		goField := field.GoName
		switch field.Desc.Kind() {
		case protoreflect.BoolKind:
			g.P("        var ", param, " C.int8_t")
			g.P("        if resp.", goField, " { ", param, " = 1 } else { ", param, " = 0 }")
		case protoreflect.StringKind:
			g.P("        var ", param, "_ptr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")))
			g.P("        var ", param, "_len C.int")
			g.P("        var ", param, "_free C.FreeFunc")
			g.P("        if len(resp.", goField, ") > 0 {")
			g.P("            ", param, "_ptr = C.CBytes([]byte(resp.", goField, "))")
			g.P("            ", param, "_len = C.int(len(resp.", goField, "))")
			g.P("            ", param, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("        }")
		case protoreflect.BytesKind:
			g.P("        var ", param, "_ptr ", g.QualifiedGoIdent(unsafePackage.Ident("Pointer")))
			g.P("        var ", param, "_len C.int")
			g.P("        var ", param, "_free C.FreeFunc")
			g.P("        if len(resp.", goField, ") > 0 {")
			g.P("            ", param, "_ptr = C.CBytes(resp.", goField, ")")
			g.P("            ", param, "_len = C.int(len(resp.", goField, "))")
			g.P("            ", param, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("        }")
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			g.P("        ", param, " := C.int32_t(resp.", goField, ")")
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			g.P("        ", param, " := C.int64_t(resp.", goField, ")")
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			g.P("        ", param, " := C.uint32_t(resp.", goField, ")")
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			g.P("        ", param, " := C.uint64_t(resp.", goField, ")")
		case protoreflect.FloatKind:
			g.P("        ", param, " := C.float(resp.", goField, ")")
		case protoreflect.DoubleKind:
			g.P("        ", param, " := C.double(resp.", goField, ")")
		default:
			g.P("        ", param, " := C.int64_t(0)")
		}
	}
	g.P("        C.", wrapper, "(onReadNative, C.uint64_t(streamHandle),")
	for _, field := range fields {
		param := nativeParamName(field, "")
		if isBytesOrStringField(field) {
			g.P("            ", param, "_ptr, ", param, "_len, ", param, "_free,")
		} else {
			g.P("            ", param, ",")
		}
	}
	g.P("        )")
	g.P("        return true")
	g.P("    }")
	g.P("    onDoneFunc := func(err error) {")
	g.P("        <-handleReady")
	g.P("        errId := uint64(0)")
	g.P("        if err != nil {")
	g.P("            errId = ", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err)")
	g.P("        }")
	g.P("        C.call_on_done(onDone, C.uint64_t(streamHandle), C.uint64_t(errId))")
	g.P("    }")
	g.P("    handle, err := ", adaptorStart, "(ctx, onRead, onDoneFunc)")
	g.P("    if err != nil {")
	g.P("        *outHandle = 0")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    streamHandle = handle")
	g.P("    close(handleReady)")
	g.P("    *outHandle = C.uint64_t(handle)")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateBidiSendNative(
	g *protogen.GeneratedFile,
	funcName string,
	reqType string,
	reqMsg *protogen.Message,
	adaptorSend string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle C.uint64_t,")
	generateNativeReqParams(g, reqMsg)
	g.P(") C.uint64_t {")
	g.P("    req := &", reqType, "{}")
	generateNativeReqAssignments(g, reqMsg)
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}

func generateBidiSendNativeTakeReq(
	g *protogen.GeneratedFile,
	funcName string,
	reqType string,
	reqMsg *protogen.Message,
	adaptorSend string,
) {
	g.P("//export ", funcName)
	g.P("func ", funcName, "(")
	g.P("    streamHandle C.uint64_t,")
	generateNativeReqParamsTakeReq(g, reqMsg)
	g.P(") C.uint64_t {")
	g.P("    req := &", reqType, "{}")
	generateNativeReqAssignmentsTakeReq(g, reqMsg)
	g.P("    if err := ", adaptorSend, "(uint64(streamHandle), req); err != nil {")
	g.P("        return C.uint64_t(", g.QualifiedGoIdent(rpcRuntimePkg.Ident("StoreError")), "(err))")
	g.P("    }")
	g.P("    return 0")
	g.P("}")
	g.P()
}
