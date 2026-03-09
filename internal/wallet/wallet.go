package wallet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
)

const (
	hardenedOffset = uint32(0x80000000)
	purposeIndex   = uint32(44) + hardenedOffset  // 44'
	coinTypeIndex  = uint32(60) + hardenedOffset   // 60' — Ethereum
)

func DerivePrivateKey(mnemonic string, account, change, addressIndex uint32) (*ecdsa.PrivateKey, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	seed := bip39.NewSeed(mnemonic, "")

	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("derive master key: %w", err)
	}

	purposeKey, err := masterKey.Derive(purposeIndex)
	if err != nil {
		return nil, fmt.Errorf("derive purpose: %w", err)
	}

	coinTypeKey, err := purposeKey.Derive(coinTypeIndex)
	if err != nil {
		return nil, fmt.Errorf("derive coin type: %w", err)
	}

	accountKey, err := coinTypeKey.Derive(account + hardenedOffset)
	if err != nil {
		return nil, fmt.Errorf("derive account: %w", err)
	}

	changeKey, err := accountKey.Derive(change)
	if err != nil {
		return nil, fmt.Errorf("derive change: %w", err)
	}

	addressKey, err := changeKey.Derive(addressIndex)
	if err != nil {
		return nil, fmt.Errorf("derive address index: %w", err)
	}

	privKeyBytes, err := addressKey.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("extract private key: %w", err)
	}

	ecdsaKey, err := crypto.ToECDSA(privKeyBytes.Serialize())
	if err != nil {
		return nil, fmt.Errorf("convert to ECDSA: %w", err)
	}

	return ecdsaKey, nil
}

func DeriveAddress(mnemonic string, account, change, addressIndex uint32) (string, error) {
	ecdsaKey, err := DerivePrivateKey(mnemonic, account, change, addressIndex)
	if err != nil {
		return "", err
	}

	address := crypto.PubkeyToAddress(ecdsaKey.PublicKey)

	return address.Hex(), nil
}

// TxParams holds parameters for an EIP-1559 transaction.
type TxParams struct {
	ChainID              *big.Int
	Nonce                uint64
	To                   common.Address
	Value                *big.Int
	GasLimit             uint64
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	Data                 []byte
}

type TxResult struct {
	SignedTx string // "0x02..." RLP-encoded hex
	TxHash   string // "0x..." keccak256 hash
}

func SignTransaction(mnemonic string, account, change, addressIndex uint32, params *TxParams) (*TxResult, error) {
	privateKey, err := DerivePrivateKey(mnemonic, account, change, addressIndex)
	if err != nil {
		return nil, fmt.Errorf("derive private key: %w", err)
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   params.ChainID,
		Nonce:     params.Nonce,
		To:        &params.To,
		Value:     params.Value,
		Gas:       params.GasLimit,
		GasTipCap: params.MaxPriorityFeePerGas,
		GasFeeCap: params.MaxFeePerGas,
		Data:      params.Data,
	})

	signer := types.NewLondonSigner(params.ChainID)

	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	rawTx, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal transaction: %w", err)
	}

	signedTxHex := "0x" + hex.EncodeToString(rawTx)
	txHash := signedTx.Hash().Hex()

	return &TxResult{
		SignedTx: signedTxHex,
		TxHash:   txHash,
	}, nil
}

func ValidateAddress(address string) (bool, string) {
	if !strings.HasPrefix(address, "0x") {
		return false, "address must start with 0x"
	}

	if len(address) != 42 {
		return false, "address must be 42 characters long (0x + 40 hex chars)"
	}

	hexPart := address[2:]
	if _, err := hex.DecodeString(hexPart); err != nil {
		return false, "address contains non-hex characters"
	}

	canonical := common.HexToAddress(address).Hex()
	if address != canonical {
		return false, fmt.Sprintf("invalid EIP-55 checksum: expected %s", canonical)
	}

	return true, ""
}
