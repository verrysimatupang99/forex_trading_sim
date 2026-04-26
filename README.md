# Forex Trading Simulator

A forex trading simulator with ML-based market prediction capabilities.

## Features

- **User Authentication** - JWT-based authentication with role management
- **Trading Accounts** - Create and manage demo trading accounts
- **Trade Execution** - Simulate buy/sell trades with margin calculations
- **Position Management** - Track open positions and calculate unrealized P&L
- **ML Predictions** - LSTM-based price prediction integration
- **Technical Indicators** - RSI, MACD, Moving Averages, Bollinger Bands
- **Backtesting** - Historical strategy testing (coming soon)

## Tech Stack

- **Backend**: Go (Golang) with Gin framework
- **Database**: PostgreSQL with GORM
- **Cache**: Redis
- **ML**: TensorFlow/PyTorch (LSTM)
- **Container**: Docker & Docker Compose

## Project Structure

```
.
├── cmd/
│   └── api/main.go           # Application entry point
├── config/
│   └── config.go             # Configuration management
├── internal/
│   ├── database/             # Database connection and migrations
│   ├── handlers/             # HTTP handlers
│   ├── middleware/           # JWT authentication middleware
│   ├── models/               # Database models
│   └── services/             # Business logic services
├── migrations/               # Database migration files
├── models/                   # ML model storage
├── docker-compose.yml        # Docker orchestration
├── Dockerfile               # Container definition
└── go.mod                   # Go dependencies
```

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- PostgreSQL (optional, using Docker)
- Redis (optional, using Docker)

### Installation

1. Clone the repository
2. Copy `.env.example` to `.env` and configure
3. Run with Docker Compose:

```bash
docker-compose up -d
```

4. The API will be available at `http://localhost:8080`

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/v1/auth/register | Register new user |
| POST | /api/v1/auth/login | Login user |
| GET | /api/v1/historical-data | Get historical prices |
| GET | /api/v1/technical-indicators | Get technical indicators |
| GET | /api/v1/users/me | Get user profile |
| POST | /api/v1/trading/accounts | Create trading account |
| POST | /api/v1/trading/trade | Execute a trade |
| GET | /api/v1/trading/positions | Get open positions |
| POST | /api/v1/predictions/predict | Get ML prediction |

## Development

### Running locally without Docker

1. Start PostgreSQL and Redis
2. Run database migrations
3. Build and run:

```bash
go mod download
go build -o main ./cmd/api
./main
```

### Running tests

```bash
go test ./...
```

## License

MIT License
