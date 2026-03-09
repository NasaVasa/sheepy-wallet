package wallet

import (
	"math/big"
	"regexp"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// knownAddress is at m/44'/60'/0'/0/0 for testMnemonic.
const knownAddress = "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"

// knownAddressAccount1 is at m/44'/60'/1'/0/0 for testMnemonic.
const knownAddressAccount1 = "0x78839F6054d7ed13918bAe0473BA31b1Ca9D7265"

var hexAddressRe = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func TestDeriveAddress_Deterministic(t *testing.T) {
	addr1, err := DeriveAddress(testMnemonic, 0, 0, 0)
	if err != nil {
		t.Fatalf("DeriveAddress returned unexpected error: %v", err)
	}

	if addr1 != knownAddress {
		t.Errorf("address mismatch: got %s, want %s", addr1, knownAddress)
	}

	addr2, err := DeriveAddress(testMnemonic, 0, 0, 0)
	if err != nil {
		t.Fatalf("DeriveAddress (second call) returned unexpected error: %v", err)
	}

	if addr1 != addr2 {
		t.Errorf("non-deterministic: first call %s, second call %s", addr1, addr2)
	}
}

func TestDeriveAddress_DifferentPaths(t *testing.T) {
	addrAccount0, err := DeriveAddress(testMnemonic, 0, 0, 0)
	if err != nil {
		t.Fatalf("account=0: %v", err)
	}

	addrAccount1, err := DeriveAddress(testMnemonic, 1, 0, 0)
	if err != nil {
		t.Fatalf("account=1: %v", err)
	}

	if addrAccount0 == addrAccount1 {
		t.Errorf("expected different addresses for account=0 and account=1, got %s for both", addrAccount0)
	}

	if addrAccount1 != knownAddressAccount1 {
		t.Errorf("account=1 address mismatch: got %s, want %s", addrAccount1, knownAddressAccount1)
	}

	addrIndex1, err := DeriveAddress(testMnemonic, 0, 0, 1)
	if err != nil {
		t.Fatalf("address_index=1: %v", err)
	}

	if addrAccount0 == addrIndex1 {
		t.Errorf("expected different addresses for address_index=0 and address_index=1, got %s for both", addrAccount0)
	}

	for _, addr := range []string{addrAccount0, addrAccount1, addrIndex1} {
		if !hexAddressRe.MatchString(addr) {
			t.Errorf("address %q does not match expected format 0x<40 hex chars>", addr)
		}
	}
}

func TestDeriveAddress_InvalidMnemonic(t *testing.T) {
	_, err := DeriveAddress("this is not a valid bip39 mnemonic phrase at all", 0, 0, 0)
	if err == nil {
		t.Fatal("expected an error for invalid mnemonic, got nil")
	}
}

func TestValidateAddress_Valid(t *testing.T) {
	valid, reason := ValidateAddress(knownAddress)
	if !valid {
		t.Errorf("expected valid=true for %s, got false (reason: %s)", knownAddress, reason)
	}
	if reason != "" {
		t.Errorf("expected empty reason for valid address, got %q", reason)
	}
}

func TestValidateAddress_NoPrefix(t *testing.T) {
	noPrefix := knownAddress[2:]
	valid, reason := ValidateAddress(noPrefix)
	if valid {
		t.Errorf("expected valid=false for address without 0x prefix")
	}
	if reason == "" {
		t.Errorf("expected non-empty reason for address without 0x prefix")
	}
}

func TestValidateAddress_WrongLength(t *testing.T) {
	cases := []string{
		"0x9858EfFD232B4033E47d90003D41EC34EcaEda",   // too short
		"0x9858EfFD232B4033E47d90003D41EC34EcaEda9400", // too long
	}
	for _, addr := range cases {
		valid, reason := ValidateAddress(addr)
		if valid {
			t.Errorf("expected valid=false for wrong-length address %q", addr)
		}
		if reason == "" {
			t.Errorf("expected non-empty reason for wrong-length address %q", addr)
		}
	}
}

func TestValidateAddress_NonHex(t *testing.T) {
	nonHex := "0x9858EfFD232B4033E47d90003D41EC34EcaEdaZZ"
	valid, reason := ValidateAddress(nonHex)
	if valid {
		t.Errorf("expected valid=false for non-hex address %q", nonHex)
	}
	if reason == "" {
		t.Errorf("expected non-empty reason for non-hex address %q", nonHex)
	}
}

func TestValidateAddress_BadChecksum(t *testing.T) {
	lowercase := strings.ToLower(knownAddress)
	valid, reason := ValidateAddress(lowercase)
	if valid {
		t.Errorf("expected valid=false for all-lowercase address %q (invalid EIP-55 checksum)", lowercase)
	}
	if reason == "" {
		t.Errorf("expected non-empty reason describing checksum mismatch for %q", lowercase)
	}
	if !strings.Contains(strings.ToLower(reason), "checksum") {
		t.Errorf("reason %q does not mention checksum", reason)
	}
}

func newBasicTxParams() *TxParams {
	return &TxParams{
		ChainID:              big.NewInt(11155111), // Sepolia
		Nonce:                0,
		To:                   common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"),
		Value:                big.NewInt(1e18),
		GasLimit:             21000,
		MaxFeePerGas:         big.NewInt(30e9),
		MaxPriorityFeePerGas: big.NewInt(1e9),
		Data:                 nil,
	}
}

func TestSignTransaction_Basic(t *testing.T) {
	params := newBasicTxParams()

	result, err := SignTransaction(testMnemonic, 0, 0, 0, params)
	if err != nil {
		t.Fatalf("SignTransaction returned unexpected error: %v", err)
	}

	if !strings.HasPrefix(result.SignedTx, "0x02") {
		t.Errorf("signed_tx must start with '0x02', got: %s", result.SignedTx[:min(len(result.SignedTx), 10)])
	}

	if !strings.HasPrefix(result.TxHash, "0x") {
		t.Errorf("tx_hash must start with '0x', got: %s", result.TxHash)
	}
	if len(result.TxHash) != 66 {
		t.Errorf("tx_hash must be 66 characters (0x + 64 hex), got %d: %s", len(result.TxHash), result.TxHash)
	}
	if !hexAddressRe.MatchString(result.TxHash[:42]) {
		hashRe := regexp.MustCompile(`^0x[0-9a-fA-F]{64}$`)
		if !hashRe.MatchString(result.TxHash) {
			t.Errorf("tx_hash is not a valid hex hash: %s", result.TxHash)
		}
	}

	result2, err := SignTransaction(testMnemonic, 0, 0, 0, params)
	if err != nil {
		t.Fatalf("SignTransaction (second call) returned unexpected error: %v", err)
	}
	if result.SignedTx != result2.SignedTx {
		t.Errorf("non-deterministic signed_tx:\n  first:  %s\n  second: %s", result.SignedTx, result2.SignedTx)
	}
	if result.TxHash != result2.TxHash {
		t.Errorf("non-deterministic tx_hash:\n  first:  %s\n  second: %s", result.TxHash, result2.TxHash)
	}
}

func TestSignTransaction_ERC20Data(t *testing.T) {
	erc20Hex := "a9059cbb000000000000000000000000d8da6bf26964af9d7eed9e03e53415d37aa960450000000000000000000000000000000000000000000000000000000000989680"
	dataBytes := make([]byte, len(erc20Hex)/2)
	for i := range dataBytes {
		hi := hexNibble(erc20Hex[i*2])
		lo := hexNibble(erc20Hex[i*2+1])
		dataBytes[i] = hi<<4 | lo
	}

	params := &TxParams{
		ChainID:              big.NewInt(11155111),
		Nonce:                1,
		To:                   common.HexToAddress("0x1c7D4B196Cb0C7B01d743Fbc6116a902379C7238"),
		Value:                big.NewInt(0),
		GasLimit:             65000,
		MaxFeePerGas:         big.NewInt(30e9),
		MaxPriorityFeePerGas: big.NewInt(1e9),
		Data:                 dataBytes,
	}

	result, err := SignTransaction(testMnemonic, 0, 0, 0, params)
	if err != nil {
		t.Fatalf("SignTransaction returned unexpected error: %v", err)
	}

	if !strings.HasPrefix(result.SignedTx, "0x02") {
		t.Errorf("signed_tx must start with '0x02' (EIP-1559 type 2), got: %s", result.SignedTx[:min(len(result.SignedTx), 10)])
	}
}

func TestSignTransaction_InvalidMnemonic(t *testing.T) {
	params := newBasicTxParams()
	_, err := SignTransaction("this is not a valid bip39 mnemonic phrase at all", 0, 0, 0, params)
	if err == nil {
		t.Fatal("expected an error for invalid mnemonic, got nil")
	}
}

func hexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		panic("invalid hex character: " + string(c))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
