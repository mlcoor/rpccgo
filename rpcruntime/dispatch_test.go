package rpcruntime

import (
	"errors"
	"sync"
	"testing"
)

func TestRegisterAndLookupGrpcHandler(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	serviceName := "rpc.test.TestService"
	handler := &struct{ name string }{name: "grpc-handler"}

	replaced, err := RegisterGrpcHandler(serviceName, handler)
	if err != nil {
		t.Fatalf("RegisterGrpcHandler failed: %v", err)
	}
	if replaced {
		t.Error("expected replaced=false for first registration")
	}

	got, ok := LookupGrpcHandler(serviceName)
	if !ok {
		t.Fatal("LookupGrpcHandler returned ok=false")
	}
	if got != handler {
		t.Error("LookupGrpcHandler returned wrong handler")
	}
}

func TestRegisterAndLookupConnectHandler(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	serviceName := "rpc.test.TestService"
	handler := &struct{ name string }{name: "connect-handler"}

	replaced, err := RegisterConnectHandler(serviceName, handler)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}
	if replaced {
		t.Error("expected replaced=false for first registration")
	}

	got, ok := LookupConnectHandler(serviceName)
	if !ok {
		t.Fatal("LookupConnectHandler returned ok=false")
	}
	if got != handler {
		t.Error("LookupConnectHandler returned wrong handler")
	}
}

func TestTwoProtocolsForSameService(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	serviceName := "rpc.test.SharedService"
	grpcHandler := &struct{ name string }{name: "grpc"}
	connectHandler := &struct{ name string }{name: "connect"}

	_, err := RegisterGrpcHandler(serviceName, grpcHandler)
	if err != nil {
		t.Fatalf("RegisterGrpcHandler failed: %v", err)
	}

	_, err = RegisterConnectHandler(serviceName, connectHandler)
	if err != nil {
		t.Fatalf("RegisterConnectHandler failed: %v", err)
	}

	gotGrpc, ok := LookupGrpcHandler(serviceName)
	if !ok {
		t.Fatal("LookupGrpcHandler returned ok=false")
	}
	if gotGrpc != grpcHandler {
		t.Error("LookupGrpcHandler returned wrong handler")
	}

	gotConnect, ok := LookupConnectHandler(serviceName)
	if !ok {
		t.Fatal("LookupConnectHandler returned ok=false")
	}
	if gotConnect != connectHandler {
		t.Error("LookupConnectHandler returned wrong handler")
	}
}

func TestReplaceRegistration(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	serviceName := "rpc.test.ReplaceService"
	handler1 := &struct{ name string }{name: "h1"}
	handler2 := &struct{ name string }{name: "h2"}

	replaced, err := RegisterGrpcHandler(serviceName, handler1)
	if err != nil {
		t.Fatalf("first RegisterGrpcHandler failed: %v", err)
	}
	if replaced {
		t.Error("expected replaced=false for first registration")
	}

	replaced, err = RegisterGrpcHandler(serviceName, handler2)
	if err != nil {
		t.Fatalf("second RegisterGrpcHandler failed: %v", err)
	}
	if !replaced {
		t.Error("expected replaced=true for second registration")
	}

	got, ok := LookupGrpcHandler(serviceName)
	if !ok {
		t.Fatal("LookupGrpcHandler returned ok=false")
	}
	if got != handler2 {
		t.Error("expected handler2 after replace")
	}
}

func TestLookupNotFound(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	_, ok := LookupGrpcHandler("rpc.test.NonExistent")
	if ok {
		t.Error("expected ok=false for non-existent service")
	}

	_, ok = LookupConnectHandler("rpc.test.NonExistent")
	if ok {
		t.Error("expected ok=false for non-existent service")
	}
}

func TestRegisterEmptyServiceName(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	handler := &struct{}{}

	_, err := RegisterGrpcHandler("", handler)
	if !errors.Is(err, ErrEmptyServiceName) {
		t.Errorf("expected ErrEmptyServiceName, got %v", err)
	}

	_, err = RegisterConnectHandler("", handler)
	if !errors.Is(err, ErrEmptyServiceName) {
		t.Errorf("expected ErrEmptyServiceName, got %v", err)
	}
}

func TestRegisterNilHandler(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	_, err := RegisterGrpcHandler("rpc.test.Service", nil)
	if !errors.Is(err, ErrNilHandler) {
		t.Errorf("expected ErrNilHandler, got %v", err)
	}

	_, err = RegisterConnectHandler("rpc.test.Service", nil)
	if !errors.Is(err, ErrNilHandler) {
		t.Errorf("expected ErrNilHandler, got %v", err)
	}
}

func TestListServices(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	_, _ = RegisterGrpcHandler("rpc.a.A", &struct{}{})
	_, _ = RegisterGrpcHandler("rpc.b.B", &struct{}{})
	_, _ = RegisterConnectHandler("rpc.c.C", &struct{}{})

	grpcServices := ListGrpcServices()
	if len(grpcServices) != 2 {
		t.Errorf("expected 2 grpc services, got %d", len(grpcServices))
	}

	connectServices := ListConnectServices()
	if len(connectServices) != 1 {
		t.Errorf("expected 1 connect service, got %d", len(connectServices))
	}
}

func TestConcurrentAccess(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	const goroutines = 100
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent registrations
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				handler := &struct{ id int }{id: id*iterations + j}
				_, _ = RegisterGrpcHandler("rpc.concurrent.Service", handler)
			}
		}(i)
	}

	// Concurrent lookups
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = LookupGrpcHandler("rpc.concurrent.Service")
			}
		}()
	}

	wg.Wait()
	// If we reach here without panic, the test passed
}

func TestConcurrentMixedOperations(t *testing.T) {
	clearHandlerRegistry()
	defer clearHandlerRegistry()

	const goroutines = 50
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 4)

	// Concurrent grpc registrations
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = RegisterGrpcHandler("rpc.mixed.Service", &struct{}{})
			}
		}(i)
	}

	// Concurrent connect registrations
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = RegisterConnectHandler("rpc.mixed.Service", &struct{}{})
			}
		}(i)
	}

	// Concurrent grpc lookups
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = LookupGrpcHandler("rpc.mixed.Service")
			}
		}()
	}

	// Concurrent connect lookups
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = LookupConnectHandler("rpc.mixed.Service")
			}
		}()
	}

	wg.Wait()
}
