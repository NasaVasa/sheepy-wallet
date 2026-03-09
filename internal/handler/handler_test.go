package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sheepy-wallet/internal/config"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func newTestHandler() *Handler {
	cfg := &config.Config{
		Config: config.ServerConfig{Host: "127.0.0.1", Port: 8000},
		Gates: []config.Gate{
			{Name: "ethereum_sepolia", Mnemonic: testMnemonic},
		},
	}
	return New(cfg)
}

func post(t *testing.T, h *Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)
	return rr
}

func get(t *testing.T, h *Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rr, req)
	return rr
}

func decodeJSON(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&m); err != nil {
		t.Fatalf("decode response JSON: %v\nbody: %s", err, rr.Body.String())
	}
	return m
}

func TestCreateAddress_Success(t *testing.T) {
	h := newTestHandler()
	body := `{"gate":"ethereum_sepolia","account":0,"change":0,"address_index":0}`
	rr := post(t, h, "/api/v1/createaddress", body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	m := decodeJSON(t, rr)
	addr, ok := m["address"].(string)
	if !ok || addr == "" {
		t.Fatalf("response does not contain non-empty 'address' field: %v", m)
	}

	if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
		t.Errorf("unexpected address format: %q", addr)
	}

	const want = "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"
	if addr != want {
		t.Errorf("address: got %q, want %q", addr, want)
	}
}

func TestCreateAddress_UnknownGate(t *testing.T) {
	h := newTestHandler()
	body := `{"gate":"nonexistent_gate","account":0,"change":0,"address_index":0}`
	rr := post(t, h, "/api/v1/createaddress", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	m := decodeJSON(t, rr)
	if _, ok := m["error"]; !ok {
		t.Errorf("expected 'error' field in response: %v", m)
	}
}

func TestCreateAddress_MethodNotAllowed(t *testing.T) {
	h := newTestHandler()
	rr := get(t, h, "/api/v1/createaddress")

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusMethodNotAllowed, rr.Body.String())
	}
}

func TestCreateAddress_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	rr := post(t, h, "/api/v1/createaddress", `{not valid json`)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	m := decodeJSON(t, rr)
	if _, ok := m["error"]; !ok {
		t.Errorf("expected 'error' field in response: %v", m)
	}
}

func newTestHandlerWithEthereum() *Handler {
	cfg := &config.Config{
		Config: config.ServerConfig{Host: "127.0.0.1", Port: 8000},
		Gates: []config.Gate{
			{Name: "ethereum_sepolia", Mnemonic: testMnemonic},
			{Name: "ethereum", Mnemonic: testMnemonic},
		},
	}
	return New(cfg)
}

func TestValidateAddress_ValidAddress(t *testing.T) {
	h := newTestHandler()
	body := `{"address":"0x9858EfFD232B4033E47d90003D41EC34EcaEda94"}`
	rr := post(t, h, "/api/v1/validateaddress", body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	m := decodeJSON(t, rr)
	valid, ok := m["valid"].(bool)
	if !ok {
		t.Fatalf("'valid' field is missing or not a bool: %v", m)
	}
	if !valid {
		t.Errorf("expected valid=true, got false (reason: %v)", m["reason"])
	}
	if reason, exists := m["reason"]; exists && reason != "" {
		t.Errorf("expected no reason for a valid address, got %q", reason)
	}
}

func TestValidateAddress_InvalidAddress(t *testing.T) {
	cases := []struct {
		name    string
		address string
	}{
		{"no_prefix", "9858EfFD232B4033E47d90003D41EC34EcaEda94"},
		{"wrong_length", "0x9858EfFD232B4033E47d90003D41EC34EcaEda"},
		{"non_hex", "0x9858EfFD232B4033E47d90003D41EC34EcaEdaZZ"},
		{"bad_checksum", "0x9858effd232b4033e47d90003d41ec34ecaeda94"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestHandler()
			body, _ := json.Marshal(map[string]string{"address": tc.address})
			rr := post(t, h, "/api/v1/validateaddress", string(body))

			if rr.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusOK, rr.Body.String())
			}

			m := decodeJSON(t, rr)
			valid, ok := m["valid"].(bool)
			if !ok {
				t.Fatalf("'valid' field is missing or not a bool: %v", m)
			}
			if valid {
				t.Errorf("expected valid=false for address %q, got true", tc.address)
			}

			reason, _ := m["reason"].(string)
			if reason == "" {
				t.Errorf("expected non-empty 'reason' for invalid address %q", tc.address)
			}
		})
	}
}

func TestSignTransaction_NativeETH(t *testing.T) {
	h := newTestHandlerWithEthereum()
	body := `{
		"gate": "ethereum",
		"account": 0,
		"change": 0,
		"address_index": 0,
		"tx_params": {
			"to": "0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045",
			"value_wei": "1000000000000000000",
			"data": "0x",
			"nonce": 0,
			"chain_id": 11155111,
			"gas_limit": 21000,
			"max_fee_per_gas_wei": "30000000000",
			"max_priority_fee_per_gas_wei": "1000000000"
		}
	}`
	rr := post(t, h, "/api/v1/tx", body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	m := decodeJSON(t, rr)

	signedTx, ok := m["signed_tx"].(string)
	if !ok || signedTx == "" {
		t.Fatalf("response does not contain non-empty 'signed_tx' field: %v", m)
	}
	if !strings.HasPrefix(signedTx, "0x02") {
		t.Errorf("signed_tx must start with '0x02' (EIP-1559 type 2), got: %s", signedTx[:min(len(signedTx), 10)])
	}

	txHash, ok := m["tx_hash"].(string)
	if !ok || txHash == "" {
		t.Fatalf("response does not contain non-empty 'tx_hash' field: %v", m)
	}
	if !strings.HasPrefix(txHash, "0x") {
		t.Errorf("tx_hash must start with '0x', got: %s", txHash)
	}
	if len(txHash) != 66 {
		t.Errorf("tx_hash must be 66 characters (0x + 64 hex), got length %d: %s", len(txHash), txHash)
	}
}

func TestSignTransaction_ERC20Transfer(t *testing.T) {
	h := newTestHandlerWithEthereum()
	erc20Data := "0xa9059cbb000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa960450000000000000000000000000000000000000000000000000000000000989680"
	body := `{
		"gate": "ethereum",
		"account": 0,
		"change": 0,
		"address_index": 0,
		"tx_params": {
			"to": "0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238",
			"value_wei": "0",
			"data": "` + erc20Data + `",
			"nonce": 1,
			"chain_id": 11155111,
			"gas_limit": 65000,
			"max_fee_per_gas_wei": "30000000000",
			"max_priority_fee_per_gas_wei": "1000000000"
		}
	}`
	rr := post(t, h, "/api/v1/tx", body)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	m := decodeJSON(t, rr)

	signedTx, ok := m["signed_tx"].(string)
	if !ok || signedTx == "" {
		t.Fatalf("response does not contain non-empty 'signed_tx' field: %v", m)
	}
	if !strings.HasPrefix(signedTx, "0x02") {
		t.Errorf("signed_tx must start with '0x02' (EIP-1559 type 2), got: %s", signedTx[:min(len(signedTx), 10)])
	}
}

func TestSignTransaction_UnknownGate(t *testing.T) {
	h := newTestHandlerWithEthereum()
	body := `{
		"gate": "nonexistent",
		"account": 0, "change": 0, "address_index": 0,
		"tx_params": {
			"to": "0x9858EfFD232B4033E47d90003D41EC34EcaEda94",
			"value_wei": "1000", "nonce": 0, "chain_id": 1,
			"gas_limit": 21000,
			"max_fee_per_gas_wei": "1000000000",
			"max_priority_fee_per_gas_wei": "100000000"
		}
	}`
	rr := post(t, h, "/api/v1/tx", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	m := decodeJSON(t, rr)
	if _, ok := m["error"]; !ok {
		t.Errorf("expected 'error' field in response: %v", m)
	}
}

func TestSignTransaction_InvalidTo(t *testing.T) {
	h := newTestHandlerWithEthereum()
	body := `{
		"gate": "ethereum",
		"account": 0, "change": 0, "address_index": 0,
		"tx_params": {
			"to": "",
			"value_wei": "1000", "nonce": 0, "chain_id": 11155111,
			"gas_limit": 21000,
			"max_fee_per_gas_wei": "30000000000",
			"max_priority_fee_per_gas_wei": "1000000000"
		}
	}`
	rr := post(t, h, "/api/v1/tx", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	m := decodeJSON(t, rr)
	if _, ok := m["error"]; !ok {
		t.Errorf("expected 'error' field in response: %v", m)
	}
}

func TestSignTransaction_InvalidValue(t *testing.T) {
	h := newTestHandlerWithEthereum()
	body := `{
		"gate": "ethereum",
		"account": 0, "change": 0, "address_index": 0,
		"tx_params": {
			"to": "0x9858EfFD232B4033E47d90003D41EC34EcaEda94",
			"value_wei": "abc", "nonce": 0, "chain_id": 11155111,
			"gas_limit": 21000,
			"max_fee_per_gas_wei": "30000000000",
			"max_priority_fee_per_gas_wei": "1000000000"
		}
	}`
	rr := post(t, h, "/api/v1/tx", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusBadRequest, rr.Body.String())
	}

	m := decodeJSON(t, rr)
	if _, ok := m["error"]; !ok {
		t.Errorf("expected 'error' field in response: %v", m)
	}
}

func TestSignTransaction_MethodNotAllowed(t *testing.T) {
	h := newTestHandlerWithEthereum()
	rr := get(t, h, "/api/v1/tx")

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d, want %d\nbody: %s", rr.Code, http.StatusMethodNotAllowed, rr.Body.String())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
