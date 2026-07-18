# HotspotOS ⚡

**HotspotOS** is a self-hosted, modular hotspot management platform and network operating system featuring Safaricom M-Pesa STK push payments, client bandwidth shaping (rate limiting), active session telemetry, and a multi-service admin control dashboard. 

It is designed to run on Ubuntu Linux, Raspberry Pi nodes, mini PCs, cloud VPSs, and local network routers.

---

## 1. Complete Architecture

```text
                     Internet
                         │
                 Ethernet / WiFi
                         │
                  HotspotOS Server
──────────────────────────────────────────────────
 Network Manager (services/network-manager)
 ├── nftables redirect chain
 ├── Client authorization sets
 ├── TC Bandwidth rate limiting
 └── Traffic telemetry simulator
──────────────────────────────────────────────────
 API Gateway (apps/api)
 ├── Fiber router / JWT Guard
 ├── Analytics aggregator
 ├── WebSocket notification server
 └── Client static portal hosting
──────────────────────────────────────────────────
 Payment Service (services/payment-service)
 ├── Daraja M-Pesa API client
 ├── Async STK Push simulator
 └── Redis Pub/Sub events
──────────────────────────────────────────────────
 Databases
 ├── Postgres (GORM domain persistence)
 └── Redis (Pub/Sub payment channel)
```

---

## 2. Directory Structure

```text
HotspotOS/
├── apps/
│   ├── api/                 # Go Fiber API gateway (Auth, Sessions, Devices, Plans, Reports)
│   ├── dashboard/           # Vite + React + TypeScript + Tailwind CSS Admin Console
│   └── mobile/              # React Native + Expo Admin Monitoring Application
├── services/
│   ├── network-manager/     # Privileged Go daemon managing nftables, leases & TC limits
│   ├── captive-portal/      # Vite + React + TS Client Splash screen served at route /
│   └── payment-service/     # Handles M-Pesa STK pushes and mock callbacks
├── packages/
│   ├── auth/                # Shared JWT & RBAC libraries
│   ├── database/            # Shared GORM Postgres connection, migrations & Redis connectors
│   ├── logger/              # Shared slog structured logging package
│   └── common/              # Shared GORM schema domain models
├── docker/                  # Service-specific Dockerfiles
└── docker-compose.yml       # Orchestrates the Postgres, Redis, and Go microservices
```

---

## 3. Getting Started (Docker Compose)

The easiest way to boot the HotspotOS stack is using Docker Compose, which spins up the database backends and mock network daemons.

### Prerequisite
Ensure you have Docker and Docker Compose installed.

### Launching the Stack
Run the following command from the root folder:

```bash
docker compose up --build
```

This will automatically:
1. Boot PostgreSQL on port `5432` and Redis on port `6379`.
2. Sync dependencies and compile the Go binaries.
3. Automatically run database migrations and seed default plans and a superuser account.
4. Build the Vite React captive portal and bundle its static production assets.
5. Host the captive portal splash screen on the gateway port at `http://localhost:8080/`.
6. Host the Vite Admin Dashboard on port `http://localhost:3000/`.

---

## 4. Default Seed Credentials

For rapid testing and development, the database migrations automatically populate:

* **Admin Username**: `admin`
* **Admin Password**: `admin123`
* **Predefined Billing Plans**:
  * **1 Hour**: `20 KES` (Bandwidth Limit: 2 Mbps Down / 1 Mbps Up)
  * **3 Hours**: `50 KES` (Bandwidth Limit: 3 Mbps Down / 1.5 Mbps Up)
  * **24 Hours**: `100 KES` (Bandwidth Limit: 5 Mbps Down / 2 Mbps Up)

---

## 5. M-Pesa STK Push Integration & Simulation

When a client connects to the WiFi node, they are redirected to the captive portal served at `http://localhost:8080/`.

1. The client enters their Safaricom phone number (e.g., `0712345678`) and selects a data plan.
2. Clicking **Pay** sends a request to the API Gateway `/payments/stk` which forwards it to the Payment Service.
3. The Payment Service registers the transaction, starts an **inactive** lease session, and fires an asynchronous simulation thread.
4. After **3 seconds** (representing the user entering their PIN on their handset), the simulator hits the callback API (`/payments/callback`) with a successful Safaricom receipt code.
5. The Payment Service flags the session as **active** in GORM and notifies the API server.
6. The API server contacts the **Network Manager** which updates the local `nftables` client sets and `tc` classes.
7. Simultaneously, the success event is published to Redis Pub/Sub, notifying the admin dashboard via WebSockets.
8. The client's captive portal transitions to the "Connected" screen, allowing them full internet access.

---

## 6. Running Tests

To verify code changes, run the integration test suite:

```bash
GOTOOLCHAIN=local go test ./...
```
This tests authentication JWT issuance, DB plans seeding, and client authorization handlers using an in-memory SQLite database.
