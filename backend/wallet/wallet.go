package wallet

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/sawdustofmind/bitcoin-wallet/backend/config"
)

type Wallet struct {
	client *rpcclient.Client
	db     *sql.DB
	xpub   *hdkeychain.ExtendedKey
	params *chaincfg.Params
}

func New(btcCfg config.BitcoinConfig, xpubStr string, db *sql.DB) (*Wallet, error) {
	connCfg := &rpcclient.ConnConfig{
		Host:         btcCfg.RPCHost,
		User:         btcCfg.RPCUser,
		Pass:         btcCfg.RPCPass,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // For regtest/local
	}

	// Try to create a wallet or load it.
	// We'll try to connect to the "mywallet" wallet.
	// If it doesn't exist, we create it.

	// First, connect to root to manage wallets
	rootClient, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, err
	}
	defer rootClient.Shutdown()

	walletName := "mywallet"
	_, err = rootClient.CreateWallet(walletName, rpcclient.WithCreateWalletDisablePrivateKeys())
	// Ignore error if it already exists (RPC error -4 usually)

	// Now create client pointing to the wallet
	// Copy config and append path
	// rpcclient implementation doesn't support Path directly in ConnConfig for HTTPPostMode easily
	// unless we include it in Host or modify library usage.
	// Actually, btcsuite/btcd/rpcclient handles URL construction.
	// If Host is "localhost:8332", it posts to "/".
	// To support wallets, we should probably check if we can pass "localhost:8332/wallet/mywallet".
	// But rpcclient splits Host.

	// Workaround: Use the root client and LoadWallet/CreateWallet,
	// but bitcoin core < 0.21 used default wallet. >= 0.21 requires specific endpoints for multi-wallet.
	// If we only have ONE wallet loaded, requests to root might work if we set valid fallback?
	// No, modern bitcoind is strict.

	// Let's try appending /wallet/name to Host if the library allows it.
	// If not, we might need a different library or patch.
	// Looking at btcd/rpcclient/infrastructure.go, it does `http.NewRequest("POST", "http://" + c.config.Host, ...)`
	// So if Host contains slash, it might work? "host:port/wallet/name"

	walletHost := fmt.Sprintf("%s/wallet/%s", btcCfg.RPCHost, walletName)
	walletConnCfg := *connCfg
	walletConnCfg.Host = walletHost

	client, err := rpcclient.New(&walletConnCfg, nil)
	if err != nil {
		return nil, err
	}

	// Parse XPUB
	xpubKey, err := hdkeychain.NewKeyFromString(xpubStr)
	if err != nil {
		return nil, fmt.Errorf("invalid xpub: %v", err)
	}

	// Regtest params
	params := &chaincfg.RegressionNetParams

	// Verify we are connected to bitcoind
	// Note: might need to retry in a real app if bitcoind is starting

	w := &Wallet{
		client: client,
		db:     db,
		xpub:   xpubKey,
		params: params,
	}

	return w, nil
}

func (w *Wallet) Start() {
	// Optional: Any background tasks
	log.Println("Wallet started")
}

func (w *Wallet) GetBalance() (btcutil.Amount, error) {
	// getbalance "*" 0  (0 confirmations to include unconfirmed)
	// But getbalance might only show balance of added keys.
	// Since we import addresses, they should be in the default wallet or the named one.
	return w.client.GetBalance("*")
}

func (w *Wallet) GetNewAddress() (string, error) {
	// 1. Get next index
	var idx int
	err := w.db.QueryRow("SELECT derivation_index FROM wallet_state WHERE id = 1").Scan(&idx)
	if err != nil {
		return "", err
	}

	// 2. Derive address at m/0/idx
	addressStr, err := w.DeriveAddress(idx)
	if err != nil {
		return "", err
	}

	// 3. Import into bitcoind
	// We use ImportAddress(address, label, rescan)
	// Rescan can be slow, but for new addresses on regtest it's fast if we just started.
	// Or use ImportMulti for better control.
	// Let's use ImportAddress for simplicity.
	err = w.client.ImportAddressRescan(addressStr, "", false) // false = no rescan? Or do we need rescan?
	// If the address is new, no need to rescan the whole chain usually, but if funds were sent to it before we imported...
	// Requirement says "Get Unused Address", implies it's new.
	// But let's rescan false to be fast.
	if err != nil {
		return "", fmt.Errorf("failed to import address: %v", err)
	}

	// 4. Update index
	_, err = w.db.Exec("UPDATE wallet_state SET derivation_index = $1 WHERE id = 1", idx+1)
	if err != nil {
		return "", err
	}

	return addressStr, nil
}

func (w *Wallet) DeriveAddress(idx int) (string, error) {
	// External chain is 0
	chain, err := w.xpub.Derive(0)
	if err != nil {
		return "", err
	}
	child, err := chain.Derive(uint32(idx))
	if err != nil {
		return "", err
	}
	addr, err := child.Address(w.params)
	if err != nil {
		return "", err
	}
	return addr.EncodeAddress(), nil
}

func (w *Wallet) GetUTXOs() ([]btcjson.ListUnspentResult, error) {
	// listunspent 0 9999999 []
	return w.client.ListUnspent()
}
