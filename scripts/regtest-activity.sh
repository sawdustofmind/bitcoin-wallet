#!/bin/bash
# Script to generate regtest activity for testing balance and UTXOs
# This creates a miner wallet, mines blocks, and sends funds to your watch-only wallet

set -e

# Load config
source "$(dirname "$0")/../config.env"

RPC_USER="${BITCOIN_RPC_USER:-user}"
RPC_PASS="${BITCOIN_RPC_PASSWORD:-password}"
RPC_PORT="${BITCOIN_RPC_PORT:-18443}"
RPC_HOST="localhost:${RPC_PORT}"

# Bitcoin CLI wrapper
btc() {
    docker exec bitcoind bitcoin-cli -regtest -rpcuser="$RPC_USER" -rpcpassword="$RPC_PASS" "$@"
}

echo "=== Regtest Activity Generator ==="
echo ""

# Create or load a miner wallet (with private keys for mining rewards)
MINER_WALLET="miner"
echo "1. Setting up miner wallet..."
if ! btc -rpcwallet="$MINER_WALLET" getwalletinfo &>/dev/null; then
    btc createwallet "$MINER_WALLET" false false "" false false 2>/dev/null || \
    btc loadwallet "$MINER_WALLET" 2>/dev/null || true
fi
echo "   Miner wallet ready"

# Get a miner address for coinbase rewards
MINER_ADDRESS=$(btc -rpcwallet="$MINER_WALLET" getnewaddress "mining" "bech32")
echo "   Miner address: $MINER_ADDRESS"

# Mine initial blocks (need 101 for coinbase maturity)
BLOCK_COUNT=$(btc getblockcount)
if [ "$BLOCK_COUNT" -lt 101 ]; then
    BLOCKS_NEEDED=$((101 - BLOCK_COUNT))
    echo ""
    echo "2. Mining $BLOCKS_NEEDED blocks for coinbase maturity..."
    btc generatetoaddress "$BLOCKS_NEEDED" "$MINER_ADDRESS" > /dev/null
    echo "   Mined $BLOCKS_NEEDED blocks"
else
    echo ""
    echo "2. Already have $BLOCK_COUNT blocks (coinbase mature)"
fi

# Get miner balance
MINER_BALANCE=$(btc -rpcwallet="$MINER_WALLET" getbalance)
echo "   Miner balance: $MINER_BALANCE BTC"

# Get an address from the watch-only wallet via the backend API
echo ""
echo "3. Getting address from your wallet via API..."
API_RESPONSE=$(curl -s http://localhost:8080/address)
WALLET_ADDRESS=$(echo "$API_RESPONSE" | jq -r '.address // empty')

if [ -z "$WALLET_ADDRESS" ]; then
    echo "   ERROR: Could not get address from backend API"
    echo "   Response was: $API_RESPONSE"
    echo "   Make sure the backend is running (make up)"
    exit 1
fi
echo "   Your wallet address: $WALLET_ADDRESS"

# Send some BTC to the watch-only wallet
AMOUNT="1.0"
echo ""
echo "4. Sending $AMOUNT BTC to your wallet..."
TXID=$(btc -rpcwallet="$MINER_WALLET" sendtoaddress "$WALLET_ADDRESS" "$AMOUNT")
echo "   Transaction ID: $TXID"

# Mine a block to confirm the transaction
echo ""
echo "5. Mining 1 block to confirm transaction..."
btc generatetoaddress 1 "$MINER_ADDRESS" > /dev/null
echo "   Transaction confirmed"

# Show results
echo ""
echo "=== Done! ==="
echo ""
echo "Your wallet should now have:"
echo "  - Address: $WALLET_ADDRESS"
echo "  - Balance: $AMOUNT BTC (confirmed)"
echo ""
echo "Test your API endpoints:"
echo "  curl http://localhost:8080/balance"
echo "  curl http://localhost:8080/utxos"
echo ""
