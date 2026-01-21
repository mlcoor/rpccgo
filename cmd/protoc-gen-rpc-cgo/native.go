package main

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func isMessageFlat(msg *protogen.Message) bool {
	for _, field := range msg.Fields {
		if field.Desc.IsMap() {
			return false
		}
		if field.Desc.IsList() {
			return false
		}
		if field.Desc.HasOptionalKeyword() {
			return false
		}
		if field.Oneof != nil && !field.Oneof.Desc.IsSynthetic() {
			return false
		}
		if field.Desc.Kind() == protoreflect.MessageKind || field.Desc.Kind() == protoreflect.GroupKind {
			return false
		}
		if field.Desc.Kind() == protoreflect.EnumKind {
			return false
		}
	}
	return true
}

func fieldToCType(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return "C.int8_t"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "C.int32_t"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "C.int64_t"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "C.uint32_t"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "C.uint64_t"
	case protoreflect.FloatKind:
		return "C.float"
	case protoreflect.DoubleKind:
		return "C.double"
	case protoreflect.StringKind, protoreflect.BytesKind:
		return "*C.char"
	default:
		return "C.int64_t"
	}
}

func fieldToGoType(field *protogen.Field) string {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64"
	case protoreflect.FloatKind:
		return "float32"
	case protoreflect.DoubleKind:
		return "float64"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BytesKind:
		return "[]byte"
	default:
		return "int64"
	}
}

func isBytesOrStringField(field *protogen.Field) bool {
	k := field.Desc.Kind()
	return k == protoreflect.StringKind || k == protoreflect.BytesKind
}

func nativeParamName(field *protogen.Field, prefix string) string {
	return prefix + strings.ToLower(string(field.Desc.Name()))
}

func generateNativeReqParams(g *protogen.GeneratedFile, msg *protogen.Message) {
	for _, field := range msg.Fields {
		ctype := fieldToCType(field)
		paramName := nativeParamName(field, "req_")
		g.P("    ", paramName, " ", ctype, ",")
		if isBytesOrStringField(field) {
			g.P("    ", paramName, "_len C.int,")
		}
	}
}

func generateNativeReqParamsTakeReq(g *protogen.GeneratedFile, msg *protogen.Message) {
	for _, field := range msg.Fields {
		ctype := fieldToCType(field)
		paramName := nativeParamName(field, "req_")
		g.P("    ", paramName, " ", ctype, ",")
		if isBytesOrStringField(field) {
			g.P("    ", paramName, "_len C.int,")
			g.P("    ", paramName, "_free C.FreeFunc,")
		}
	}
}

func generateNativeRespParams(g *protogen.GeneratedFile, msg *protogen.Message) {
	for _, field := range msg.Fields {
		ctype := fieldToCType(field)
		paramName := nativeParamName(field, "resp_")

		if isBytesOrStringField(field) {
			// string/bytes are returned as malloc buffers + a free function pointer.
			g.P("    ", paramName, " **C.char,")
			g.P("    ", paramName, "_len *C.int,")
			g.P("    ", paramName, "_free *C.FreeFunc,")
		} else {
			g.P("    ", paramName, " *", ctype, ",")
		}
	}
}

func generateNativeReqAssignments(g *protogen.GeneratedFile, msg *protogen.Message) {
	for _, field := range msg.Fields {
		paramName := nativeParamName(field, "req_")
		goField := field.GoName

		switch field.Desc.Kind() {
		case protoreflect.BoolKind:
			g.P("    req.", goField, " = ", paramName, " != 0")
		case protoreflect.StringKind:
			g.P("    req.", goField, " = C.GoStringN(", paramName, ", ", paramName, "_len)")
		case protoreflect.BytesKind:
			g.P(
				"    req.",
				goField,
				" = C.GoBytes(",
				g.QualifiedGoIdent(unsafePackage.Ident("Pointer")),
				"(",
				paramName,
				"), ",
				paramName,
				"_len)",
			)
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			g.P("    req.", goField, " = int32(", paramName, ")")
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			g.P("    req.", goField, " = int64(", paramName, ")")
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			g.P("    req.", goField, " = uint32(", paramName, ")")
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			g.P("    req.", goField, " = uint64(", paramName, ")")
		case protoreflect.FloatKind:
			g.P("    req.", goField, " = float32(", paramName, ")")
		case protoreflect.DoubleKind:
			g.P("    req.", goField, " = float64(", paramName, ")")
		}
	}
}

func generateNativeReqAssignmentsTakeReq(g *protogen.GeneratedFile, msg *protogen.Message) {
	for _, field := range msg.Fields {
		paramName := nativeParamName(field, "req_")
		goField := field.GoName

		switch field.Desc.Kind() {
		case protoreflect.BoolKind:
			g.P("    req.", goField, " = ", paramName, " != 0")
		case protoreflect.StringKind:
			g.P("    req.", goField, " = C.GoStringN(", paramName, ", ", paramName, "_len)")
			g.P("    if ", paramName, "_free != nil {")
			g.P(
				"        C.call_free_func(",
				paramName,
				"_free, ",
				g.QualifiedGoIdent(unsafePackage.Ident("Pointer")),
				"(",
				paramName,
				"))",
			)
			g.P("    }")
		case protoreflect.BytesKind:
			g.P(
				"    req.",
				goField,
				" = C.GoBytes(",
				g.QualifiedGoIdent(unsafePackage.Ident("Pointer")),
				"(",
				paramName,
				"), ",
				paramName,
				"_len)",
			)
			g.P("    if ", paramName, "_free != nil {")
			g.P(
				"        C.call_free_func(",
				paramName,
				"_free, ",
				g.QualifiedGoIdent(unsafePackage.Ident("Pointer")),
				"(",
				paramName,
				"))",
			)
			g.P("    }")
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			g.P("    req.", goField, " = int32(", paramName, ")")
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			g.P("    req.", goField, " = int64(", paramName, ")")
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			g.P("    req.", goField, " = uint32(", paramName, ")")
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			g.P("    req.", goField, " = uint64(", paramName, ")")
		case protoreflect.FloatKind:
			g.P("    req.", goField, " = float32(", paramName, ")")
		case protoreflect.DoubleKind:
			g.P("    req.", goField, " = float64(", paramName, ")")
		}
	}
}

func generateNativeRespAssignments(g *protogen.GeneratedFile, msg *protogen.Message) {
	for _, field := range msg.Fields {
		paramName := nativeParamName(field, "resp_")
		goField := field.GoName

		switch field.Desc.Kind() {
		case protoreflect.BoolKind:
			g.P("    if resp.", goField, " {")
			g.P("        *", paramName, " = 1")
			g.P("    } else {")
			g.P("        *", paramName, " = 0")
			g.P("    }")
		case protoreflect.StringKind:
			g.P("    if len(resp.", goField, ") > 0 {")
			g.P("        buf := C.CBytes([]byte(resp.", goField, "))")
			g.P("        *", paramName, " = (*C.char)(buf)")
			g.P("        *", paramName, "_len = C.int(len(resp.", goField, "))")
			g.P("        *", paramName, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("    } else {")
			g.P("        *", paramName, " = nil")
			g.P("        *", paramName, "_len = 0")
			g.P("        *", paramName, "_free = nil")
			g.P("    }")
		case protoreflect.BytesKind:
			g.P("    if len(resp.", goField, ") > 0 {")
			g.P("        buf := C.CBytes(resp.", goField, ")")
			g.P("        *", paramName, " = (*C.char)(buf)")
			g.P("        *", paramName, "_len = C.int(len(resp.", goField, "))")
			g.P("        *", paramName, "_free = (C.FreeFunc)(C.Ygrpc_Free)")
			g.P("    } else {")
			g.P("        *", paramName, " = nil")
			g.P("        *", paramName, "_len = 0")
			g.P("        *", paramName, "_free = nil")
			g.P("    }")
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
			g.P("    *", paramName, " = C.int32_t(resp.", goField, ")")
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			g.P("    *", paramName, " = C.int64_t(resp.", goField, ")")
		case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			g.P("    *", paramName, " = C.uint32_t(resp.", goField, ")")
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			g.P("    *", paramName, " = C.uint64_t(resp.", goField, ")")
		case protoreflect.FloatKind:
			g.P("    *", paramName, " = C.float(resp.", goField, ")")
		case protoreflect.DoubleKind:
			g.P("    *", paramName, " = C.double(resp.", goField, ")")
		}
	}
}
