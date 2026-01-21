// Package rpcruntime provides small runtime helpers intended to be used by
// generated CGO code.
//
// Currently it contains a global error message registry:
//   - Go code stores an error message and receives an integer errorId.
//   - C code retrieves the message through a stable ABI (Ygrpc_GetErrorMsg)
//     implemented in the generated CGO package main.
//
// The registry retains records for approximately 3 seconds and then treats them
// as expired.
package rpcruntime
