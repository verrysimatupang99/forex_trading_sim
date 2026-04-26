#!/bin/bash

# ============================================
# FOREX TRADING SIMULATOR - SETUP SCRIPT
# ============================================

echo "========================================"
echo "Forex Trading Simulator - Setup"
echo "========================================"

# Check OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     OS_NAME="linux";;
    Darwin*)    OS_NAME="macos";;
    CYGWIN*)    OS_NAME="windows";;
    MINGW*)     OS_NAME="windows";;
    *)          OS_NAME="unknown";;
esac

echo "Detected OS: ${OS_NAME}"

# ============================================
# 1. INSTALL GO
# ============================================
echo ""
echo "========================================"
echo "[1/5] Installing Go..."
echo "========================================"

if command -v go &> /dev/null; then
    echo "Go already installed: $(go version)"
else
    case "${OS_NAME}" in
        linux)
            # Download and install Go
            cd /tmp
            wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
            sudo rm -rf /usr/local/go
            sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
            export PATH=$PATH:/usr/local/go/bin
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
            ;;
        macos)
            brew install go
            ;;
        windows)
            # Download and install Go from https://go.dev/dl/
            echo "Please download Go from: https://go.dev/dl/go1.21.6.windows-amd64.msi"
            ;;
    esac
fi

# ============================================
# 2. INSTALL DOCKER
# ============================================
echo ""
echo "========================================"
echo "[2/5] Installing Docker..."
echo "========================================"

if command -v docker &> /dev/null; then
    echo "Docker already installed: $(docker --version)"
else
    case "${OS_NAME}" in
        linux)
            echo "Install Docker:"
            echo "  curl -fsSL https://get.docker.com -o get-docker.sh"
            echo "  sudo sh get-docker.sh"
            echo "  sudo usermod -aG docker \$USER"
            ;;
        macos)
            echo "Download Docker Desktop: https://www.docker.com/products/docker-desktop"
            ;;
    esac
fi

# ============================================
# 3. CLONE PROJECT
# ============================================
echo ""
echo "========================================"
echo "[3/5] Project Setup"
echo "========================================"

# Create project directory
mkdir -p ~/forex-trading-sim
cd ~/forex-trading-sim

echo "Project directory: $(pwd)"

# Copy project files (if not already cloned)
if [ ! -f "docker-compose.yml" ]; then
    echo "Please copy project files to this directory"
fi

# ============================================
# 4. CREATE ENVIRONMENT FILE
# ============================================
echo ""
echo "========================================"
echo "[4/5] Creating Environment File..."
echo "========================================"

if [ ! -f ".env" ]; then
    cat > .env << 'EOF'
# Server
SERVER_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=forex_sim

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT - WAJIB DIGANTI!
JWT_SECRET=your-super-secure-secret-key-min-32-characters

# Forex Data (Frankfurter - Gratis!)
DATA_SOURCE=frankfurter

# Optional: Database Pooling
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10

# Security
BCRYPT_COST=12
EOF
    echo ".env file created!"
else
    echo ".env file already exists"
fi

# ============================================
# 5. RUN THE APPLICATION
# ============================================
echo ""
echo "========================================"
echo "[5/5] Running Application"
echo "========================================"

echo ""
echo "To start the application:"
echo "  docker-compose up -d"
echo ""
echo "Or without Docker:"
echo "  go mod download"
echo "  go build -o main ./cmd/api"
echo "  ./main"
echo ""
echo "========================================"
echo "Setup Complete!"
echo "========================================"
echo ""
echo "Next steps:"
echo "1. Edit .env and change JWT_SECRET"
echo "2. Run: docker-compose up -d"
echo "3. Test: curl http://localhost:8080/health"
echo ""
