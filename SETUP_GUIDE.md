# 🚀 Forex Trading Simulator - Quick Setup Guide

## Sistem Minimum
- **OS**: Windows 10+, macOS, atau Ubuntu 20.04+
- **RAM**: 4GB
- **Storage**: 10GB free
- **Port**: 8080, 5432, 6379 tersedia

---

## ⚡ Cara Cepat (3 Langkah)

### Langkah 1: Install Go + Docker

**Windows:**
```powershell
# Install Chocolatey (jika belum)
Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Install Go
choco install golang -y

# Install Docker
choco install docker-desktop -y
```

**macOS:**
```bash
# Install Homebrew (jika belum)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Go dan Docker
brew install go
brew install --cask docker
```

**Linux (Ubuntu):**
```bash
# Install Go
sudo apt update
sudo apt install golang-go

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
# Logout dan login ulang
```

---

### Langkah 2: Clone/Copy Project

```bash
# Clone dari Git (jika ada)
git clone <repo-url> forex-trading-sim
cd forex-trading-sim

# Atau copy folder project ke komputer
```

---

### Langkah 3: Setup & Run

```bash
# 1. Copy environment file
cp .env.example .env

# 2. Edit JWT_SECRET di .env (WAJIB!)
nano .env
# atau
code .env

# 3. Run dengan Docker
docker-compose up -d

# 4. Cek status
docker-compose ps
```

---

## ✅ Verifikasi Installation

```bash
# Health check
curl http://localhost:8080/health

# Response harusnya:
# {"status":"ok","timestamp":"2025-..."}

# Cek currency pairs
curl http://localhost:8080/api/v1/currency-pairs
```

---

## 🔧 Jika Tanpa Docker

```bash
# Install PostgreSQL & Redis
# Ubuntu:
sudo apt install postgresql redis-server

# Start services
sudo systemctl start postgresql
sudo systemctl start redis-server

# Setup database
sudo -u postgres createdb forex_sim
sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';"

# Run Go app
go mod download
go build -o main ./cmd/api
./main
```

---

## 📦 Environment Variables

| Variable | Default | Keterangan |
|----------|---------|------------|
| `JWT_SECRET` | - | **WAJIB** diganti! |
| `DATA_SOURCE` | frankfurter | forex data API |
| `DB_HOST` | localhost | PostgreSQL host |
| `SERVER_PORT` | 8080 | API port |

---

## 🐛 Troubleshooting

### Port sudah terpakai
```bash
# Cek port
lsof -i :8080

# Kill process
kill -9 <PID>
```

### Database connection error
```bash
# Cek PostgreSQL
sudo systemctl status postgresql

# Restart
sudo systemctl restart postgresql
```

### Go module error
```bash
go mod tidy
go clean -modcache
go mod download
```

---

## 📱 API Endpoints

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/health` | Health check |
| GET | `/api/v1/currency-pairs` | List pairs (public) |
| POST | `/api/v1/auth/register` | Daftar user |
| POST | `/api/v1/auth/login` | Login |
| GET | `/api/v1/trading/accounts` | Akun trading |
| POST | `/api/v1/trading/trade` | Eksekusi trade |

---

## 🎯 Contoh Penggunaan

```bash
# 1. Daftar user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email":"test@example.com",
    "password":"Password123",
    "first_name":"John",
    "last_name":"Doe"
  }'

# 2. Login (dapat token)
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Password123"}'

# 3. Buat akun trading (pakai token)
curl -X POST http://localhost:8080/api/v1/trading/accounts \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"balance":10000,"leverage":100,"currency":"USD"}'
```

---

## 📞 Support

Jika ada masalah, cek:
1. `docker-compose logs -f api` - lihat logs
2. `docker-compose ps` - cek status container
3. Pastikan port 8080, 5432, 6379 kosong

---

**Happy Trading! 📈**
