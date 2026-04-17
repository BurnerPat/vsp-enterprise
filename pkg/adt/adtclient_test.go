package adt

import (
	"context"
	"net/http"
	"testing"

	"github.com/oisee/vibing-steampunk/pkg/adt/connection"
)

// --------------------------------------------------------------------------
// mockAdtConnection implements connection.AdtConnection for testing.
// --------------------------------------------------------------------------

type mockAdtConnection struct {
	sendFn  func(ctx context.Context, req *connection.Request) (*connection.AdtResponse, error)
	pingErr error
	closed  bool
}

func (m *mockAdtConnection) SendRequest(ctx context.Context, req *connection.Request) (*connection.AdtResponse, error) {
	if m.sendFn != nil {
		return m.sendFn(ctx, req)
	}
	return connection.NewAdtResponseFromMap(200, nil, []byte("mock")), nil
}
func (m *mockAdtConnection) Ping(_ context.Context) error { return m.pingErr }
func (m *mockAdtConnection) Close() error                 { m.closed = true; return nil }

// --------------------------------------------------------------------------
// AdtClient interface tests via NewClientWithConnection
// --------------------------------------------------------------------------

func TestNewClientWithConnection_ImplementsAdtClient(t *testing.T) {
	cfg := NewConfig("https://sap.example.com", "user", "pass")
	conn := &mockAdtConnection{}

	var client ClientInterface = NewClientWithConnection(cfg, conn)
	if client == nil {
		t.Fatal("NewClientWithConnection returned nil")
	}
}

func TestClient_SendRequest_ViaNativeConnection(t *testing.T) {
	cfg := NewConfig("https://sap.example.com", "user", "pass")
	conn := &mockAdtConnection{
		sendFn: func(_ context.Context, req *connection.Request) (*connection.AdtResponse, error) {
			return connection.NewAdtResponseFromMap(200, nil, []byte("native-response")), nil
		},
	}
	client := NewClientWithConnection(cfg, conn)

	resp, err := client.SendRequest(context.Background(), &connection.Request{
		Path:   "/sap/bc/adt/test",
		Method: http.MethodGet,
	})
	if err != nil {
		t.Fatalf("SendRequest: %v", err)
	}
	if string(resp.Body) != "native-response" {
		t.Errorf("Body = %q, want native-response", string(resp.Body))
	}
}

func TestClient_Close_ClosesConnection(t *testing.T) {
	cfg := NewConfig("https://sap.example.com", "user", "pass")
	conn := &mockAdtConnection{}
	client := NewClientWithConnection(cfg, conn)

	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !conn.closed {
		t.Error("Close should have called connection.Close()")
	}
}

func TestClient_Connect_PingsConnection(t *testing.T) {
	cfg := NewConfig("https://sap.example.com", "user", "pass")
	conn := &mockAdtConnection{}
	client := NewClientWithConnection(cfg, conn)

	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
}

func TestClient_GetConfig(t *testing.T) {
	cfg := NewConfig("https://sap.example.com", "user", "pass", WithClient("100"))
	client := NewClientWithConnection(cfg, &mockAdtConnection{})

	if client.GetConfig().Client != "100" {
		t.Errorf("GetConfig().Client = %q, want 100", client.GetConfig().Client)
	}
}

func TestClient_Connection(t *testing.T) {
	conn := &mockAdtConnection{}
	client := NewClientWithConnection(NewConfig("https://sap.example.com", "user", "pass"), conn)

	if client.Connection() != conn {
		t.Error("Connection() should return the injected connection")
	}
}

// --------------------------------------------------------------------------
// Legacy Client (NewClient) still works — backward compatibility
// --------------------------------------------------------------------------

func TestClient_LegacyClient_SendRequest(t *testing.T) {
	// Legacy client created without an AdtConnection should still support SendRequest
	// via the legacy Requester path.
	cfg := NewConfig("https://sap.example.com", "user", "pass")
	mockHTTP := &mockTransportClient{
		responses: map[string]*http.Response{
			"/sap/bc/adt/test": newTestResponse("legacy-body"),
		},
	}
	legacyTransport := NewTransportWithClient(cfg, mockHTTP)
	client := NewClientWithTransport(cfg, legacyTransport)

	// SendRequest should work through the legacy fallback path.
	resp, err := client.SendRequest(context.Background(), &connection.Request{
		Path:   "/sap/bc/adt/test",
		Method: http.MethodGet,
	})
	if err != nil {
		t.Fatalf("SendRequest on legacy client: %v", err)
	}
	if string(resp.Body) != "legacy-body" {
		t.Errorf("Body = %q, want legacy-body", string(resp.Body))
	}
}

func TestClient_LegacyClient_ConnectionIsNil(t *testing.T) {
	client := NewClient("https://sap.example.com", "user", "pass")
	if client.Connection() != nil {
		t.Error("Legacy client should have nil Connection()")
	}
}

// --------------------------------------------------------------------------
// connectionRequesterAdapter tests
// --------------------------------------------------------------------------

func TestConnectionRequesterAdapter_BridgesNewToLegacy(t *testing.T) {
	conn := &mockAdtConnection{
		sendFn: func(_ context.Context, req *connection.Request) (*connection.AdtResponse, error) {
			return connection.NewAdtResponseFromMap(200, map[string]string{
				"X-Custom": "header-val",
			}, []byte("bridged")), nil
		},
	}
	client := NewClientWithConnection(NewConfig("https://sap.example.com", "user", "pass"), conn)

	// Use legacy path (c.connection.Request) by calling an existing Client method
	// that hasn't been migrated yet.
	source, err := client.GetProgram(context.Background(), "ZTEST")
	if err != nil {
		t.Fatalf("GetProgram via bridge: %v", err)
	}
	if source != "bridged" {
		t.Errorf("source = %q, want bridged", source)
	}
}
