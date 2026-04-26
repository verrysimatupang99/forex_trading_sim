document.addEventListener('DOMContentLoaded', () => {
    // DOM Elements
    const loginView = document.getElementById('login-view');
    const dashboardView = document.getElementById('dashboard-view');
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    const tradeForm = document.getElementById('trade-form');
    const predictionForm = document.getElementById('prediction-form');
    const backtestForm = document.getElementById('backtest-form');
    
    // Global store for currency pairs
    let currencyPairs = [];

    // Check authentication
    if (auth.isAuthenticated()) {
        showDashboard();
    } else {
        showLogin();
    }

    // Login Form
    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('email').value;
        const password = document.getElementById('password').value;

        try {
            await auth.login(email, password);
            auth.setUserEmail(email);
            showDashboard();
        } catch (error) {
            showMessage('login-form', error.message, 'error');
        }
    });

    // Register Form
    registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('reg-email').value;
        const password = document.getElementById('reg-password').value;
        const name = document.getElementById('reg-name').value;

        try {
            await auth.register(email, password, name);
            auth.setUserEmail(email);
            showDashboard();
        } catch (error) {
            showMessage('register-form', error.message, 'error');
        }
    });

    // Toggle Login/Register forms
    document.getElementById('show-register').addEventListener('click', (e) => {
        e.preventDefault();
        loginForm.style.display = 'none';
        registerForm.style.display = 'block';
    });

    document.getElementById('show-login').addEventListener('click', (e) => {
        e.preventDefault();
        registerForm.style.display = 'none';
        loginForm.style.display = 'block';
    });

    // Logout
    document.getElementById('logout-btn').addEventListener('click', () => {
        auth.logout();
        showLogin();
    });

    // Navigation
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const view = btn.dataset.view;
            switchView(view);
            
            // Load data for the view
            if (view === 'home') loadHomeData();
            if (view === 'trade') loadTradeData();
            if (view === 'positions') loadPositions();
            if (view === 'history') loadHistory();
            if (view === 'predictions') loadPredictionsData();
            if (view === 'backtest') loadBacktestData();
        });
    });

    // Trade Form
    tradeForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const pairId = document.getElementById('trade-pair').value;
        const type = document.getElementById('trade-type').value;
        const amount = document.getElementById('trade-amount').value;
        const price = document.getElementById('trade-price').value;
        const accountId = auth.getAccountId();
        
        // Find the currency pair string from the selected ID
        const selectedPair = currencyPairs.find(p => p.id == pairId);
        const pairString = selectedPair ? `${selectedPair.base_currency}/${selectedPair.quote_currency}` : pairId;

        try {
            const result = await api.executeTrade(accountId, pairString, type, amount, price);
            showMessage('trade-result', `Trade executed successfully! Trade ID: ${result.trade?.id || 'N/A'}`, 'success');
            tradeForm.reset();
        } catch (error) {
            showMessage('trade-result', error.message, 'error');
        }
    });

    // Prediction Form
    predictionForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const pairId = document.getElementById('pred-pair').value;
        const periods = document.getElementById('pred-periods').value;
        
        // Find the currency pair string from the selected ID
        const selectedPair = currencyPairs.find(p => p.id == pairId);
        const pairString = selectedPair ? `${selectedPair.base_currency}/${selectedPair.quote_currency}` : pairId;

        try {
            const result = await api.predict(pairString, periods);
            showMessage('prediction-result', `Prediction: ${JSON.stringify(result.prediction || result)}`, 'success');
        } catch (error) {
            showMessage('prediction-result', error.message, 'error');
        }
    });

    // Backtest Form
    backtestForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const pairId = document.getElementById('bt-pair').value;
        const strategy = document.getElementById('bt-strategy').value;
        const startDate = document.getElementById('bt-start').value;
        const endDate = document.getElementById('bt-end').value;
        const initialCapital = document.getElementById('bt-initial').value;

        try {
            const result = await api.runBacktest(pairId, strategy, startDate, endDate, initialCapital);
            showMessage('backtest-result', `Backtest completed! Result ID: ${result.result?.id || 'N/A'}`, 'success');
        } catch (error) {
            showMessage('backtest-result', error.message, 'error');
        }
    });

    // Helper Functions
    function showLogin() {
        loginView.style.display = 'block';
        dashboardView.style.display = 'none';
    }

    function showDashboard() {
        loginView.style.display = 'none';
        dashboardView.style.display = 'block';
        document.getElementById('user-email').textContent = auth.getUserEmail();
        loadHomeData();
    }

    function switchView(viewName) {
        // Update nav buttons
        document.querySelectorAll('.nav-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.view === viewName);
        });

        // Hide all sections
        document.querySelectorAll('.content-section').forEach(section => {
            section.style.display = 'none';
        });

        // Show selected section
        const section = document.getElementById(`${viewName}-section`);
        if (section) {
            section.style.display = 'block';
        }
    }

    function showMessage(formId, message, type) {
        const container = document.getElementById(formId).parentElement;
        let msgDiv = container.querySelector('.result-message');
        
        if (!msgDiv) {
            msgDiv = document.createElement('div');
            msgDiv.className = 'result-message';
            container.appendChild(msgDiv);
        }
        
        msgDiv.className = `result-message ${type}`;
        msgDiv.textContent = message;
        
        setTimeout(() => {
            msgDiv.remove();
        }, 5000);
    }

    async function loadHomeData() {
        try {
            const accountId = auth.getAccountId();
            
            // Get accounts
            const accounts = await api.getAccounts();
            if (accounts && accounts.length > 0) {
                document.getElementById('account-id').textContent = accounts[0].id;
                document.getElementById('account-balance').textContent = `$${accounts[0].balance?.toFixed(2) || '0.00'}`;
            }

            // Get positions count
            if (accountId) {
                const positions = await api.getPositions(accountId);
                document.getElementById('open-positions-count').textContent = positions?.length || 0;
            }

            // Get trade history count
            if (accountId) {
                const history = await api.getTradeHistory(accountId);
                document.getElementById('total-trades-count').textContent = history?.length || 0;
            }

            // Load currency pairs
            const pairs = await api.getCurrencyPairs();
            const container = document.getElementById('currency-pairs-list');
            container.innerHTML = '';
            
            // Handle different API response formats
            let pairList = [];
            if (Array.isArray(pairs)) {
                pairList = pairs;
            } else if (pairs.data && Array.isArray(pairs.data)) {
                pairList = pairs.data;
            } else if (pairs.currency_pairs && Array.isArray(pairs.currency_pairs)) {
                pairList = pairs.currency_pairs;
            }
            
            pairList.forEach(pair => {
                const div = document.createElement('div');
                div.className = 'currency-pair-card';
                div.innerHTML = `
                    <div class="pair">${pair.base_currency || pair.base}/${pair.quote_currency || pair.quote}</div>
                    <div class="rate">${pair.rate?.toFixed(5) || 'N/A'}</div>
                `;
                container.appendChild(div);
            });

        } catch (error) {
            console.error('Error loading home data:', error);
        }
    }

    async function loadTradeData() {
        try {
            const pairs = await api.getCurrencyPairs();
            
            // Handle different API response formats
            if (Array.isArray(pairs)) {
                currencyPairs = pairs;
            } else if (pairs.data && Array.isArray(pairs.data)) {
                currencyPairs = pairs.data;
            } else if (pairs.currency_pairs && Array.isArray(pairs.currency_pairs)) {
                currencyPairs = pairs.currency_pairs;
            } else {
                currencyPairs = [];
            }
            
            const select = document.getElementById('trade-pair');
            const predSelect = document.getElementById('pred-pair');
            const btSelect = document.getElementById('bt-pair');
            
            // Clear existing options (keep first)
            select.innerHTML = '<option value="">Select Pair</option>';
            predSelect.innerHTML = '<option value="">Select Pair</option>';
            btSelect.innerHTML = '<option value="">Select Pair</option>';
            
            currencyPairs.forEach(pair => {
                const displayName = `${pair.base_currency || pair.base}/${pair.quote_currency || pair.quote}`;
                const option = `<option value="${pair.id}">${displayName}</option>`;
                select.innerHTML += option;
                predSelect.innerHTML += option;
                btSelect.innerHTML += option;
            });

            // Set default dates for backtest
            const today = new Date();
            const lastMonth = new Date(today);
            lastMonth.setMonth(lastMonth.getMonth() - 1);
            
            document.getElementById('bt-end').value = today.toISOString().split('T')[0];
            document.getElementById('bt-start').value = lastMonth.toISOString().split('T')[0];

        } catch (error) {
            console.error('Error loading trade data:', error);
        }
    }

    async function loadPositions() {
        try {
            const accountId = auth.getAccountId();
            const positions = await api.getPositions(accountId);
            const tbody = document.getElementById('positions-tbody');
            tbody.innerHTML = '';

            const posList = positions || [];
            if (posList.length === 0) {
                tbody.innerHTML = '<tr><td colspan="8" style="text-align: center;">No open positions</td></tr>';
                return;
            }

            posList.forEach(pos => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${pos.id}</td>
                    <td>${pos.currency_pair}</td>
                    <td>${pos.type}</td>
                    <td>${pos.amount}</td>
                    <td>${pos.entry_price?.toFixed(5) || 'N/A'}</td>
                    <td>${pos.current_price?.toFixed(5) || pos.entry_price?.toFixed(5) || 'N/A'}</td>
                    <td class="${pos.profit_loss >= 0 ? 'profit' : 'loss'}">${pos.profit_loss?.toFixed(2) || '0.00'}</td>
                    <td><button class="btn-danger" onclick="closePosition(${pos.id})">Close</button></td>
                `;
                tbody.appendChild(tr);
            });

        } catch (error) {
            console.error('Error loading positions:', error);
            document.getElementById('positions-tbody').innerHTML = 
                `<tr><td colspan="8" style="text-align: center; color: #dc3545;">${error.message}</td></tr>`;
        }
    }

    async function loadHistory() {
        try {
            const accountId = auth.getAccountId();
            const history = await api.getTradeHistory(accountId);
            const tbody = document.getElementById('history-tbody');
            tbody.innerHTML = '';

            const tradeList = history || [];
            if (tradeList.length === 0) {
                tbody.innerHTML = '<tr><td colspan="7" style="text-align: center;">No trade history</td></tr>';
                return;
            }

            tradeList.forEach(trade => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${trade.id}</td>
                    <td>${trade.currency_pair}</td>
                    <td>${trade.type}</td>
                    <td>${trade.amount}</td>
                    <td>${trade.price?.toFixed(5) || 'N/A'}</td>
                    <td>${trade.closed_at || 'Open'}</td>
                    <td class="${trade.profit_loss >= 0 ? 'profit' : 'loss'}">${trade.profit_loss?.toFixed(2) || '0.00'}</td>
                `;
                tbody.appendChild(tr);
            });

        } catch (error) {
            console.error('Error loading history:', error);
            document.getElementById('history-tbody').innerHTML = 
                `<tr><td colspan="7" style="text-align: center; color: #dc3545;">${error.message}</td></tr>`;
        }
    }

    async function loadPredictionsData() {
        await loadTradeData(); // Load currency pairs for prediction
    }

    async function loadBacktestData() {
        await loadTradeData(); // Load currency pairs for backtest
    }

    // Global function to close position
    window.closePosition = async function(positionId) {
        if (!confirm('Are you sure you want to close this position?')) return;
        
        try {
            await api.closePosition(positionId);
            loadPositions(); // Reload positions
            loadHomeData(); // Refresh home data
        } catch (error) {
            alert('Error closing position: ' + error.message);
        }
    };
});
