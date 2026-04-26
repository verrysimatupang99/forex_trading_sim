const API_BASE = '/api/v1';

class ApiService {
    constructor() {
        this.token = localStorage.getItem('token');
    }

    setToken(token) {
        this.token = token;
        if (token) {
            localStorage.setItem('token', token);
        } else {
            localStorage.removeItem('token');
        }
    }

    getToken() {
        return this.token || localStorage.getItem('token');
    }

    async request(endpoint, options = {}) {
        const url = `${API_BASE}${endpoint}`;
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers
        };

        const token = this.getToken();
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        try {
            const response = await fetch(url, {
                ...options,
                headers
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.message || 'Request failed');
            }

            return data;
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    }

    // Auth endpoints
    async login(email, password) {
        return this.request('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
    }

    async register(email, password, name) {
        return this.request('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ email, password, name })
        });
    }

    // Currency pairs (public)
    async getCurrencyPairs() {
        return this.request('/currency-pairs');
    }

    // User
    async getProfile() {
        return this.request('/users/me');
    }

    // Trading - Accounts
    async getAccounts() {
        return this.request('/trading/accounts');
    }

    async createAccount(initialBalance = 10000) {
        return this.request('/trading/accounts', {
            method: 'POST',
            body: JSON.stringify({ initial_balance: initialBalance })
        });
    }

    async getBalance(accountId) {
        return this.request(`/trading/accounts/${accountId}/balance`);
    }

    // Trading - Execute Trade
    async executeTrade(accountId, pair, tradeType, amount, price) {
        return this.request('/trading/trade', {
            method: 'POST',
            body: JSON.stringify({
                account_id: accountId,
                currency_pair: pair,
                type: tradeType,
                quantity: parseFloat(amount),
                entry_price: parseFloat(price)
            })
        });
    }

    // Trading - Positions
    async getPositions(accountId) {
        return this.request(`/trading/positions?account_id=${accountId}`);
    }

    async closePosition(positionId) {
        return this.request(`/trading/positions/${positionId}`, {
            method: 'DELETE'
        });
    }

    // Trading - Trade History
    async getTradeHistory(accountId) {
        return this.request(`/trading/trades?account_id=${accountId}`);
    }

    // Predictions
    async predict(pair, periods = 10) {
        return this.request('/predictions/predict', {
            method: 'POST',
            body: JSON.stringify({ currency_pair: pair, periods })
        });
    }

    async getPredictionHistory(accountId) {
        return this.request(`/predictions/history?account_id=${accountId}`);
    }

    // Backtest
    async runBacktest(currencyPairId, strategyName, startDate, endDate, initialCapital = 10000) {
        return this.request('/backtest/run', {
            method: 'POST',
            body: JSON.stringify({
                currency_pair_id: parseInt(currencyPairId),
                strategy_name: strategyName,
                timeframe: "1h",
                start_date: startDate,
                end_date: endDate,
                initial_capital: initialCapital,
                commission: 0,
                slippage_pips: 0,
                spread_pips: 2
            })
        });
    }

    async getBacktestResults(accountId) {
        return this.request(`/backtest/results?account_id=${accountId}`);
    }

    async getBacktestResult(resultId) {
        return this.request(`/backtest/results/${resultId}`);
    }

    // Historical Data (public)
    async getHistoricalData(pair, startDate, endDate) {
        return this.request(`/historical-data?pair=${pair}&start_date=${startDate}&end_date=${endDate}`);
    }

    // Technical Indicators (public)
    async getTechnicalIndicators(pair, indicator = 'SMA', period = 14) {
        return this.request(`/technical-indicators?pair=${pair}&indicator=${indicator}&period=${period}`);
    }
}

const api = new ApiService();
