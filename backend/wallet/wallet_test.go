package wallet

import (
	"database/sql"
	"testing"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

func TestDeriveAddress(t *testing.T) {
	// Generate a fresh master key for testing
	seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	if err != nil {
		t.Fatalf("Failed to generate seed: %v", err)
	}
	master, err := hdkeychain.NewMaster(seed, &chaincfg.RegressionNetParams)
	if err != nil {
		t.Fatalf("Failed to create master key: %v", err)
	}
	neuter, err := master.Neuter()
	if err != nil {
		t.Fatalf("Failed to neuter key: %v", err)
	}
	xpubStr := neuter.String()

	xpubKey, err := hdkeychain.NewKeyFromString(xpubStr)
	if err != nil {
		t.Fatalf("Failed to parse xpub: %v", err)
	}

	w := &Wallet{
		xpub:   xpubKey,
		params: &chaincfg.RegressionNetParams,
		db:     &sql.DB{}, // Mock or nil, not used in DeriveAddress
	}

	// Test index 0
	addr0, err := w.DeriveAddress(0)
	if err != nil {
		t.Fatalf("Failed to derive address 0: %v", err)
	}
	t.Logf("Address 0: %s", addr0)

	// Test index 1
	addr1, err := w.DeriveAddress(1)
	if err != nil {
		t.Fatalf("Failed to derive address 1: %v", err)
	}
	t.Logf("Address 1: %s", addr1)

	if addr0 == addr1 {
		t.Fatal("Addresses should be different")
	}
}
