package ton

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInvalidDestination  = errors.New("transaction destination does not match")
	ErrInsufficientAmount  = errors.New("transaction amount is insufficient")
	ErrInvalidBOC          = errors.New("invalid BOC format")
)

type Verifier struct {
	testnet       bool
	walletAddress string
	client        ton.APIClientWrapped
}

func NewVerifier(testnet bool, walletAddress string) *Verifier {
	return &Verifier{
		testnet:       testnet,
		walletAddress: walletAddress,
	}
}

// TransactionInfo contains verified transaction details
type TransactionInfo struct {
	Hash        string
	FromAddress string
	ToAddress   string
	Amount      uint64 // in nanoTON
	Comment     string
	Timestamp   uint32
}

// connect establishes connection to TON network
func (v *Verifier) connect(ctx context.Context) error {
	if v.client != nil {
		return nil
	}

	client := liteclient.NewConnectionPool()

	// Use public config
	configURL := "https://ton.org/global.config.json"
	if v.testnet {
		configURL = "https://ton.org/testnet-global.config.json"
	}

	err := client.AddConnectionsFromConfigUrl(ctx, configURL)
	if err != nil {
		return fmt.Errorf("failed to connect to TON network: %w", err)
	}

	v.client = ton.NewAPIClient(client).WithRetry()
	return nil
}

// VerifyTransaction verifies a TON transaction from BOC
// The BOC is the signed transaction result from TON Connect
func (v *Verifier) VerifyTransaction(boc string, expectedAmountNano int64, expectedComment string) (*TransactionInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("[TON] Verifying transaction, expected amount: %d nano\n", expectedAmountNano)

	// Connect to network
	if err := v.connect(ctx); err != nil {
		return nil, err
	}

	// Parse our wallet address
	walletAddr, err := address.ParseAddr(v.walletAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid wallet address: %w", err)
	}

	// Get recent transactions to our wallet
	txs, err := v.getRecentTransactions(ctx, walletAddr, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	fmt.Printf("[TON] Found %d recent transactions to wallet\n", len(txs))

	// Look for matching transaction
	now := uint32(time.Now().Unix())
	for _, tx := range txs {
		// Skip if too old (more than 10 minutes)
		if now-tx.Timestamp > 600 {
			continue
		}

		// Check amount
		if int64(tx.Amount) < expectedAmountNano-1000000 { // 0.001 TON tolerance
			continue
		}

		fmt.Printf("[TON] Found matching transaction: hash=%s, amount=%d, from=%s\n",
			tx.Hash, tx.Amount, tx.FromAddress)
		return &tx, nil
	}

	// If not found in recent transactions, try to parse BOC directly
	// to extract transaction hash and verify
	if boc != "" {
		txInfo, err := v.verifyFromBOC(ctx, boc, walletAddr, expectedAmountNano)
		if err == nil {
			return txInfo, nil
		}
		fmt.Printf("[TON] BOC verification failed: %v\n", err)
	}

	return nil, ErrTransactionNotFound
}

// getRecentTransactions fetches recent incoming transactions to the wallet
func (v *Verifier) getRecentTransactions(ctx context.Context, addr *address.Address, limit int) ([]TransactionInfo, error) {
	// Get account state
	master, err := v.client.CurrentMasterchainInfo(ctx)
	if err != nil {
		return nil, err
	}

	account, err := v.client.GetAccount(ctx, master, addr)
	if err != nil {
		return nil, err
	}

	if !account.IsActive {
		return nil, nil
	}

	// Get transactions
	txs, err := v.client.ListTransactions(ctx, addr, uint32(limit), account.LastTxLT, account.LastTxHash)
	if err != nil {
		return nil, err
	}

	var result []TransactionInfo
	for _, tx := range txs {
		// We only care about incoming internal messages
		if tx.IO.In == nil {
			continue
		}

		// Safely try to get internal message (AsInternal panics on external)
		txInfo, ok := tryParseInternalMessage(tx, addr)
		if !ok {
			continue
		}

		result = append(result, txInfo)
	}

	return result, nil
}

// verifyFromBOC tries to extract and verify transaction from BOC
func (v *Verifier) verifyFromBOC(ctx context.Context, bocStr string, expectedDest *address.Address, expectedAmount int64) (*TransactionInfo, error) {
	// Decode BOC
	bocBytes, err := base64.StdEncoding.DecodeString(bocStr)
	if err != nil {
		// Try URL-safe base64
		bocBytes, err = base64.URLEncoding.DecodeString(bocStr)
		if err != nil {
			return nil, ErrInvalidBOC
		}
	}

	// Parse cell
	c, err := cell.FromBOC(bocBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BOC: %w", err)
	}

	// The BOC from TON Connect contains the external message
	// We need to extract the transaction details
	// For now, we'll just log and return not found since
	// the proper way is to track the transaction on-chain

	fmt.Printf("[TON] Parsed BOC cell: %x\n", c.Hash())

	// The transaction should appear in the wallet's transaction list
	// after it's processed by the network. We've already checked that above.
	return nil, ErrTransactionNotFound
}

// tryParseInternalMessage safely parses an internal message, returning false if it's external
func tryParseInternalMessage(tx *tlb.Transaction, addr *address.Address) (info TransactionInfo, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			// It's an external message, not internal
			ok = false
		}
	}()

	inMsg := tx.IO.In.AsInternal()
	if inMsg == nil {
		return TransactionInfo{}, false
	}

	// Extract comment from message body if any
	comment := ""
	if inMsg.Body != nil {
		comment = extractComment(inMsg.Body)
	}

	// Get sender address
	fromAddr := ""
	if inMsg.SrcAddr != nil {
		fromAddr = inMsg.SrcAddr.String()
	}

	return TransactionInfo{
		Hash:        base64.StdEncoding.EncodeToString(tx.Hash),
		FromAddress: fromAddr,
		ToAddress:   addr.String(),
		Amount:      inMsg.Amount.Nano().Uint64(),
		Comment:     comment,
		Timestamp:   tx.Now,
	}, true
}

// extractComment extracts text comment from message body
func extractComment(body *cell.Cell) string {
	if body == nil {
		return ""
	}

	slice := body.BeginParse()

	// Check for text comment (op = 0)
	op, err := slice.LoadUInt(32)
	if err != nil {
		return ""
	}

	if op == 0 {
		// Text comment
		data, err := slice.LoadBinarySnake()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}

	return ""
}
