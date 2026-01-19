package service

import (
	"context"
	"fmt"
	"time"

	"github.com/zyvpn/backend/internal/model"
	"github.com/zyvpn/backend/internal/repository"
	"github.com/zyvpn/backend/internal/ton"
)

const (
	TonWorkerInterval   = 10 * time.Second // Check every 10 seconds
	TonPaymentTimeout   = 10 * time.Minute // Fail payments older than 10 minutes
)

type TonWorker struct {
	repo        *repository.Repository
	verifier    *ton.Verifier
	balanceSvc  *BalanceService
	paymentSvc  *PaymentService
}

func NewTonWorker(
	repo *repository.Repository,
	verifier *ton.Verifier,
	balanceSvc *BalanceService,
	paymentSvc *PaymentService,
) *TonWorker {
	return &TonWorker{
		repo:       repo,
		verifier:   verifier,
		balanceSvc: balanceSvc,
		paymentSvc: paymentSvc,
	}
}

// Start begins the background worker
func (w *TonWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(TonWorkerInterval)
	defer ticker.Stop()

	fmt.Println("[TON Worker] Started, checking every", TonWorkerInterval)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[TON Worker] Stopped")
			return
		case <-ticker.C:
			w.processAwaitingPayments(ctx)
		}
	}
}

// processAwaitingPayments checks all payments waiting for tx confirmation
func (w *TonWorker) processAwaitingPayments(ctx context.Context) {
	payments, err := w.repo.GetAwaitingTxPayments(ctx)
	if err != nil {
		fmt.Printf("[TON Worker] Error getting awaiting payments: %v\n", err)
		return
	}

	if len(payments) == 0 {
		return
	}

	fmt.Printf("[TON Worker] Processing %d awaiting payments\n", len(payments))

	for _, payment := range payments {
		w.processPayment(ctx, &payment)
	}
}

// processPayment tries to verify a single payment
func (w *TonWorker) processPayment(ctx context.Context, payment *model.Payment) {
	// Check if payment is too old
	if time.Since(payment.CreatedAt) > TonPaymentTimeout {
		fmt.Printf("[TON Worker] Payment %s timed out, marking as failed\n", payment.ID)
		w.repo.UpdatePaymentStatus(ctx, payment.ID, model.PaymentStatusFailed)
		return
	}

	// Get expected amount in nanoTON
	expectedAmountNano := int64(payment.Amount * 1e9)

	// Try to find the transaction
	txInfo, err := w.verifier.VerifyTransaction("", expectedAmountNano, "")
	if err != nil {
		// Transaction not found yet - keep waiting
		fmt.Printf("[TON Worker] Payment %s: transaction not found yet\n", payment.ID)
		return
	}

	fmt.Printf("[TON Worker] Payment %s: found transaction hash=%s, amount=%d\n",
		payment.ID, txInfo.Hash, txInfo.Amount)

	// Update external ID with transaction hash
	if err := w.repo.UpdatePaymentExternalID(ctx, payment.ID, txInfo.Hash); err != nil {
		fmt.Printf("[TON Worker] Error updating external ID: %v\n", err)
		return
	}

	// Complete the payment based on type
	if payment.PaymentType == model.PaymentTypeTopUp {
		// Credit balance
		tonAmount := payment.Amount
		_, err = w.balanceSvc.CreditTopUp(ctx, payment.UserID, tonAmount, payment.ID)
		if err != nil {
			fmt.Printf("[TON Worker] Error crediting balance: %v\n", err)
			return
		}
		// Update status
		if err := w.repo.UpdatePaymentStatus(ctx, payment.ID, model.PaymentStatusCompleted); err != nil {
			fmt.Printf("[TON Worker] Error updating status: %v\n", err)
			return
		}
		fmt.Printf("[TON Worker] Payment %s completed (top-up %.4f TON)\n", payment.ID, tonAmount)
	} else {
		// Subscription payment - use payment service
		if err := w.paymentSvc.CompletePayment(ctx, payment.ID); err != nil {
			fmt.Printf("[TON Worker] Error completing payment: %v\n", err)
			return
		}
		fmt.Printf("[TON Worker] Payment %s completed (subscription)\n", payment.ID)
	}
}
