package handler

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"sheepy-wallet/internal/config"
	"sheepy-wallet/internal/wallet"
)

type Handler struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/createaddress", h.CreateAddress)
	mux.HandleFunc("/api/v1/validateaddress", h.ValidateAddress)
	mux.HandleFunc("/api/v1/tx", h.SignTransaction)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type createAddressRequest struct {
	Gate         string `json:"gate"`
	Account      uint32 `json:"account"`
	Change       uint32 `json:"change"`
	AddressIndex uint32 `json:"address_index"`
}

type createAddressResponse struct {
	Address string `json:"address"`
}

func (h *Handler) CreateAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req createAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	gate, err := h.cfg.FindGate(req.Gate)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	address, err := wallet.DeriveAddress(gate.Mnemonic, req.Account, req.Change, req.AddressIndex)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "address derivation failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, createAddressResponse{Address: address})
}

type validateAddressRequest struct {
	Gate    string `json:"gate"`
	Address string `json:"address"`
}

type validateAddressResponse struct {
	Valid  bool   `json:"valid"`
	Reason string `json:"reason,omitempty"`
}

func (h *Handler) ValidateAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req validateAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	valid, reason := wallet.ValidateAddress(req.Address)

	writeJSON(w, http.StatusOK, validateAddressResponse{
		Valid:  valid,
		Reason: reason,
	})
}

type signTransactionRequest struct {
	Gate         string   `json:"gate"`
	Account      uint32   `json:"account"`
	Change       uint32   `json:"change"`
	AddressIndex uint32   `json:"address_index"`
	TxParams     txParams `json:"tx_params"`
}

type txParams struct {
	To                      string `json:"to"`
	ValueWei                string `json:"value_wei"`
	Data                    string `json:"data"`
	Nonce                   uint64 `json:"nonce"`
	ChainID                 uint64 `json:"chain_id"`
	GasLimit                uint64 `json:"gas_limit"`
	MaxFeePerGasWei         string `json:"max_fee_per_gas_wei"`
	MaxPriorityFeePerGasWei string `json:"max_priority_fee_per_gas_wei"`
}

type signTransactionResponse struct {
	SignedTx string `json:"signed_tx"`
	TxHash   string `json:"tx_hash"`
}

func (h *Handler) SignTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req signTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	gate, err := h.cfg.FindGate(req.Gate)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tp := req.TxParams

	if tp.To == "" || !strings.HasPrefix(tp.To, "0x") || len(tp.To) != 42 {
		writeError(w, http.StatusBadRequest, "invalid 'to' address: must be a 0x-prefixed 40-character hex string")
		return
	}

	value, ok := new(big.Int).SetString(tp.ValueWei, 10)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid 'value_wei': must be a decimal integer string")
		return
	}

	maxFeePerGas, ok := new(big.Int).SetString(tp.MaxFeePerGasWei, 10)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid 'max_fee_per_gas_wei': must be a decimal integer string")
		return
	}

	maxPriorityFeePerGas, ok := new(big.Int).SetString(tp.MaxPriorityFeePerGasWei, 10)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid 'max_priority_fee_per_gas_wei': must be a decimal integer string")
		return
	}

	// "0x" and "" are treated as empty/nil
	var dataBytes []byte
	trimmed := strings.TrimPrefix(tp.Data, "0x")
	if trimmed != "" {
		dataBytes, err = hex.DecodeString(trimmed)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid 'data': not valid hex: "+err.Error())
			return
		}
	}

	params := wallet.TxParams{
		ChainID:              new(big.Int).SetUint64(tp.ChainID),
		Nonce:                tp.Nonce,
		To:                   common.HexToAddress(tp.To),
		Value:                value,
		GasLimit:             tp.GasLimit,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
		Data:                 dataBytes,
	}

	result, err := wallet.SignTransaction(gate.Mnemonic, req.Account, req.Change, req.AddressIndex, &params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "sign transaction failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, signTransactionResponse{
		SignedTx: result.SignedTx,
		TxHash:   result.TxHash,
	})
}
