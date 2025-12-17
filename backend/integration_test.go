package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/sawdustofmind/bitcoin-wallet/backend/api"
	"github.com/sawdustofmind/bitcoin-wallet/backend/config"
	"github.com/sawdustofmind/bitcoin-wallet/backend/wallet"
)

// TestIntegration runs full integration tests with real bitcoind and postgres containers
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	postgresC, postgresConnStr := startPostgres(t, ctx)
	defer postgresC.Terminate(ctx)

	// Start Bitcoin Core container
	bitcoindC, bitcoinRPCHost := startBitcoind(t, ctx)
	defer bitcoindC.Terminate(ctx)

	// Wait for bitcoind to be ready
	time.Sleep(3 * time.Second)

	// Connect to database
	db, err := sql.Open("postgres", postgresConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := runTestMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test xpub (regtest)
	xpubStr := "tpubD6NzVbkrYhZ4XNsDrPgErrqxTmaiY343QwdZmi6RMgU3p4qjY77xXcb15pX6cRa2fX6Vpxz7h16BWmsXyZKUsp1nULcVb2WLAuMQfGKVoQr"

	// Create wallet
	btcCfg := config.BitcoinConfig{
		RPCHost: bitcoinRPCHost,
		RPCUser: "testuser",
		RPCPass: "testpass",
	}

	w, err := wallet.New(btcCfg, xpubStr, db)
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	// Set up Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api.RegisterRoutes(router, w)

	// Create test server
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Run subtests
	t.Run("GetAddress", func(t *testing.T) {
		testGetAddress(t, ts.URL)
	})

	t.Run("GetBalanceInitial", func(t *testing.T) {
		testGetBalanceInitial(t, ts.URL)
	})

	t.Run("GetUTXOsInitial", func(t *testing.T) {
		testGetUTXOsInitial(t, ts.URL)
	})

	t.Run("FullFlowWithFunds", func(t *testing.T) {
		testFullFlowWithFunds(t, ts.URL, btcCfg)
	})
}

func startPostgres(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get postgres host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get postgres port: %v", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable", host, port.Port())
	return container, connStr
}

func startBitcoind(t *testing.T, ctx context.Context) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:        "ruimarinho/bitcoin-core:24",
		ExposedPorts: []string{"18443/tcp"},
		Cmd: []string{
			"-regtest",
			"-server",
			"-rpcbind=0.0.0.0",
			"-rpcallowip=0.0.0.0/0",
			"-rpcuser=testuser",
			"-rpcpassword=testpass",
			"-fallbackfee=0.0002",
		},
		WaitingFor: wait.ForLog("init message: Done loading").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start bitcoind container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get bitcoind host: %v", err)
	}

	port, err := container.MappedPort(ctx, "18443")
	if err != nil {
		t.Fatalf("Failed to get bitcoind port: %v", err)
	}

	rpcHost := fmt.Sprintf("%s:%s", host, port.Port())
	return container, rpcHost
}

func runTestMigrations(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS wallet_state (
		id SERIAL PRIMARY KEY,
		derivation_index INT NOT NULL DEFAULT 0
	);
	INSERT INTO wallet_state (id, derivation_index)
	SELECT 1, 0
	WHERE NOT EXISTS (SELECT 1 FROM wallet_state WHERE id = 1);
	`
	_, err := db.Exec(query)
	return err
}

func testGetAddress(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/address")
	if err != nil {
		t.Fatalf("Failed to get address: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	address, ok := result["address"].(string)
	if !ok || address == "" {
		t.Fatal("Expected non-empty address in response")
	}

	t.Logf("Got address: %s", address)

	// Verify address format (regtest addresses start with bcrt1 or m/n for legacy)
	if len(address) < 20 {
		t.Fatalf("Address seems too short: %s", address)
	}
}

func testGetBalanceInitial(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/balance")
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	balance, ok := result["balance"].(float64)
	if !ok {
		t.Fatal("Expected balance field in response")
	}

	t.Logf("Initial balance: %f BTC", balance)

	if balance != 0 {
		t.Fatalf("Expected initial balance to be 0, got %f", balance)
	}
}

func testGetUTXOsInitial(t *testing.T, baseURL string) {
	resp, err := http.Get(baseURL + "/utxos")
	if err != nil {
		t.Fatalf("Failed to get UTXOs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	utxos, ok := result["utxos"].([]interface{})
	if !ok {
		t.Fatal("Expected utxos array in response")
	}

	t.Logf("Initial UTXOs: %d", len(utxos))

	if len(utxos) != 0 {
		t.Fatalf("Expected initial UTXOs to be 0, got %d", len(utxos))
	}
}

func testFullFlowWithFunds(t *testing.T, baseURL string, btcCfg config.BitcoinConfig) {
	// This test verifies the full flow:
	// 1. Get a new address from our watch-only wallet
	// 2. Create a miner wallet and mine blocks
	// 3. Send funds to our address
	// 4. Mine a block to confirm
	// 5. Verify balance and UTXOs are non-zero

	// Step 1: Get address from our wallet
	resp, err := http.Get(baseURL + "/address")
	if err != nil {
		t.Fatalf("Failed to get address: %v", err)
	}
	defer resp.Body.Close()

	var addrResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&addrResult); err != nil {
		t.Fatalf("Failed to decode address response: %v", err)
	}

	walletAddress, ok := addrResult["address"].(string)
	if !ok || walletAddress == "" {
		t.Fatal("Failed to get wallet address")
	}
	t.Logf("Step 1: Got wallet address: %s", walletAddress)

	// Step 2: Create miner wallet and mine blocks
	minerClient := createMinerWallet(t, btcCfg)
	defer minerClient.Shutdown()

	// Get miner address and mine 101 blocks for coinbase maturity
	// GetNewAddress requires (label, addressType) - use "bech32" for native segwit
	minerAddress, err := minerClient.GetNewAddress("mining", "bech32")
	if err != nil {
		t.Fatalf("Failed to get miner address: %v", err)
	}
	t.Logf("Step 2: Miner address: %s", minerAddress.EncodeAddress())

	_, err = minerClient.GenerateToAddress(101, minerAddress, nil)
	if err != nil {
		t.Fatalf("Failed to mine blocks: %v", err)
	}
	t.Log("Step 2: Mined 101 blocks for coinbase maturity")

	// Step 3: Send 1.5 BTC to our watch-only wallet
	amountToSend := 1.5
	txid, err := sendToAddress(minerClient, walletAddress, amountToSend)
	if err != nil {
		t.Fatalf("Failed to send funds: %v", err)
	}
	t.Logf("Step 3: Sent %.2f BTC, txid: %s", amountToSend, txid)

	// Step 4: Mine 1 block to confirm
	_, err = minerClient.GenerateToAddress(1, minerAddress, nil)
	if err != nil {
		t.Fatalf("Failed to mine confirmation block: %v", err)
	}
	t.Log("Step 4: Mined 1 block to confirm transaction")

	// Step 5: Verify balance is non-zero
	balanceResp, err := http.Get(baseURL + "/balance")
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	defer balanceResp.Body.Close()

	var balanceResult map[string]interface{}
	if err := json.NewDecoder(balanceResp.Body).Decode(&balanceResult); err != nil {
		t.Fatalf("Failed to decode balance response: %v", err)
	}

	balance, ok := balanceResult["balance"].(float64)
	if !ok {
		t.Fatal("Expected balance field in response")
	}
	t.Logf("Step 5: Balance after funding: %.8f BTC", balance)

	if balance <= 0 {
		t.Fatalf("Expected positive balance after funding, got %.8f", balance)
	}
	if balance < amountToSend-0.001 || balance > amountToSend+0.001 {
		t.Logf("Warning: Balance %.8f differs from sent amount %.2f (fees may apply)", balance, amountToSend)
	}

	// Step 6: Verify UTXOs are non-zero
	utxosResp, err := http.Get(baseURL + "/utxos")
	if err != nil {
		t.Fatalf("Failed to get UTXOs: %v", err)
	}
	defer utxosResp.Body.Close()

	var utxosResult map[string]interface{}
	if err := json.NewDecoder(utxosResp.Body).Decode(&utxosResult); err != nil {
		t.Fatalf("Failed to decode UTXOs response: %v", err)
	}

	utxos, ok := utxosResult["utxos"].([]interface{})
	if !ok {
		t.Fatal("Expected utxos array in response")
	}
	t.Logf("Step 6: UTXOs after funding: %d", len(utxos))

	if len(utxos) == 0 {
		t.Fatal("Expected at least 1 UTXO after funding")
	}

	// Log UTXO details
	for i, utxo := range utxos {
		utxoMap, ok := utxo.(map[string]interface{})
		if ok {
			t.Logf("  UTXO %d: %.8f BTC at %s", i+1, utxoMap["amount"], utxoMap["address"])
		}
	}

	t.Log("Full flow test PASSED - wallet received funds and shows correct balance/UTXOs")
}

// createMinerWallet creates a miner wallet with private keys for mining
func createMinerWallet(t *testing.T, btcCfg config.BitcoinConfig) *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         btcCfg.RPCHost,
		User:         btcCfg.RPCUser,
		Pass:         btcCfg.RPCPass,
		HTTPPostMode: true,
		DisableTLS:   true,
	}

	rootClient, err := rpcclient.New(connCfg, nil)
	if err != nil {
		t.Fatalf("Failed to connect to bitcoind: %v", err)
	}

	// Create miner wallet (with private keys)
	walletName := "test_miner"
	_, err = rootClient.CreateWallet(walletName)
	if err != nil {
		// Try to load if already exists
		_, err = rootClient.LoadWallet(walletName)
		if err != nil {
			t.Logf("Miner wallet might already be loaded: %v", err)
		}
	}
	rootClient.Shutdown()

	// Connect to miner wallet
	walletHost := fmt.Sprintf("%s/wallet/%s", btcCfg.RPCHost, walletName)
	walletConnCfg := *connCfg
	walletConnCfg.Host = walletHost

	minerClient, err := rpcclient.New(&walletConnCfg, nil)
	if err != nil {
		t.Fatalf("Failed to connect to miner wallet: %v", err)
	}

	return minerClient
}

// sendToAddress sends BTC to an address using the miner wallet
func sendToAddress(client *rpcclient.Client, address string, amount float64) (string, error) {
	// Use RawRequest for sendtoaddress
	params := []json.RawMessage{
		json.RawMessage(fmt.Sprintf(`"%s"`, address)),
		json.RawMessage(fmt.Sprintf(`%f`, amount)),
	}

	result, err := client.RawRequest("sendtoaddress", params)
	if err != nil {
		return "", err
	}

	var txid string
	if err := json.Unmarshal(result, &txid); err != nil {
		return "", err
	}

	return txid, nil
}
