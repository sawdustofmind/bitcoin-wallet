# Bitcoin Wallet Application

## Overview

A simple Bitcoin wallet application built with Golang, React, PostgreSQL, and Bitcoin Core (regtest).

## Prerequisites

- Docker
- Docker Compose
- Make (optional, for convenience)

## Running the Application

### Using Makefile (Recommended)

- **Start everything**:
  ```bash
  make up-d
  ```

- **Run Tests**:
  ```bash
  make test
  ```

- **Stop**:
  ```bash
  make down
  ```

- **View Logs**:
  ```bash
  make logs
  ```

### Manual Method

1. Start the services:
   ```bash
   docker compose up --build
   ```

2. Access the application:
   - Frontend: [http://localhost:3000](http://localhost:3000)
   - Backend API: [http://localhost:8080](http://localhost:8080)

3. Run fixtures with wallet activity:
   ```bash
   make regtest-activity
   ```

## Configuration

The wallet uses an Extended Public Key (XPUB) configured in `docker-compose.yml`.
Default XPUB: `tpubD6NzVbkrYhZ4XNsDrPgErrqxTmaiY343QwdZmi6RMgU3p4qjY77xXcb15pX6cRa2fX6Vpxz7h16BWmsXyZKUsp1nULcVb2WLAuMQfGKVoQr`

## Testing

### Unit Tests
To run the backend unit tests:

```bash
cd backend
go test ./...
```
(Requires Go installed locally)

### Integration Tests
To run tests inside the container:

```bash
docker compose exec backend go test ./...
```

## API Endpoints

- `GET /balance`: Returns wallet balance.
- `GET /address`: Generates a new receive address.
- `GET /utxos`: Lists unspent transaction outputs.

## Design Decisions

- **Architecture**: Separated Backend (Go), Frontend (React), DB (Postgres), and Node (Bitcoind).
- **Wallet Logic**: 
  - Uses `bitcoind` as the source of truth for UTXOs and Balance.
  - Manages address derivation index in PostgreSQL.
  - Derives addresses from XPUB in Go and imports them into `bitcoind` as watch-only addresses using `importaddress`.
  - Uses a named wallet "mywallet" in `bitcoind` to segregate data.
- **Frontend**: Minimal React UI to demonstrate functionality.
- **Database**: Stores only the derivation index.
