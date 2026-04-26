#!/bin/bash
export SERVER_PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=mrtrickster99
export DB_PASSWORD=postgres
export DB_NAME=forex_sim
export REDIS_HOST=localhost
export REDIS_PORT=6379
export JWT_SECRET=forex-sim-secret-key-2024-secure-32chars
export DATA_SOURCE=frankfurter
cd /home/mrtrickster99/Documents/Coding/forex_trading_sim
exec ./main
