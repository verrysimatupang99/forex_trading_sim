class AuthService {
    constructor() {
        this.currentUser = null;
        this.accountId = null;
    }

    async login(email, password) {
        try {
            const response = await api.login(email, password);
            const token = response.token || response.access_token;
            if (token) {
                api.setToken(token);
                this.currentUser = response.user || { email };
                
                // Get or create account
                await this.initializeAccount();
                
                return response;
            }
            throw new Error('No token received');
        } catch (error) {
            console.error('Login error:', error);
            throw error;
        }
    }

    async register(email, password, name) {
        try {
            const response = await api.register(email, password, name);
            const token = response.token || response.access_token;
            if (token) {
                api.setToken(token);
                this.currentUser = { email, name };
                
                // Create account for new user
                await this.createAccount(10000);
                
                return response;
            }
            throw new Error('No token received');
        } catch (error) {
            console.error('Register error:', error);
            throw error;
        }
    }

    async initializeAccount() {
        try {
            const accounts = await api.getAccounts();
            if (accounts && accounts.length > 0) {
                this.accountId = accounts[0].id;
            } else {
                await this.createAccount(10000);
            }
        } catch (error) {
            console.error('Error initializing account:', error);
            // Try to create account
            await this.createAccount(10000);
        }
    }

    async createAccount(initialBalance) {
        try {
            const response = await api.createAccount(initialBalance);
            if (response.account) {
                this.accountId = response.account.id;
            } else if (response.id) {
                this.accountId = response.id;
            }
            return response;
        } catch (error) {
            console.error('Error creating account:', error);
            throw error;
        }
    }

    logout() {
        api.setToken(null);
        this.currentUser = null;
        this.accountId = null;
        localStorage.removeItem('token');
        localStorage.removeItem('user_email');
    }

    isAuthenticated() {
        return !!api.getToken();
    }

    getAccountId() {
        return this.accountId;
    }

    getUserEmail() {
        return this.currentUser?.email || localStorage.getItem('user_email');
    }

    setUserEmail(email) {
        this.currentUser = { ...this.currentUser, email };
        localStorage.setItem('user_email', email);
    }
}

const auth = new AuthService();
