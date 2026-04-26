const { chromium } = require('playwright');

(async () => {
  console.log('Starting browser test...');
  
  const browser = await chromium.launch({ 
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });
  const context = await browser.newContext();
  const page = await context.newPage();
  
  // Listen for console messages
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log('Console Error:', msg.text());
    }
  });

  try {
    // 1. Go to the web app
    console.log('\n=== Testing: Load Web App ===');
    await page.goto('http://localhost:8080/', { waitUntil: 'networkidle', timeout: 30000 });
    console.log('✅ Page loaded successfully');
    
    // Check if login form is visible
    const loginForm = await page.locator('#login-form').isVisible();
    console.log('✅ Login form visible:', loginForm);
    
    // 2. Test Login
    console.log('\n=== Testing: Login ===');
    await page.fill('#email', 'user1@test.com');
    await page.fill('#password', 'User1234');
    await page.click('button[type="submit"]');
    
    // Wait for dashboard to load
    await page.waitForSelector('#dashboard-view', { timeout: 10000 });
    console.log('✅ Login successful - Dashboard loaded');
    
    // 3. Check Dashboard
    console.log('\n=== Testing: Dashboard ===');
    const userEmail = await page.locator('#user-email').textContent();
    console.log('✅ User email displayed:', userEmail);
    
    const accountId = await page.locator('#account-id').textContent();
    console.log('✅ Account ID:', accountId);
    
    const balance = await page.locator('#account-balance').textContent();
    console.log('✅ Balance:', balance);
    
    // 4. Check Currency Pairs
    console.log('\n=== Testing: Currency Pairs ===');
    await page.waitForSelector('.currency-pair-card', { timeout: 5000 });
    const pairCount = await page.locator('.currency-pair-card').count();
    console.log('✅ Currency pairs displayed:', pairCount);
    
    // 5. Test Navigation - Trade
    console.log('\n=== Testing: Trade Page ===');
    await page.click('[data-view="trade"]');
    await page.waitForSelector('#trade-form', { timeout: 5000 });
    console.log('✅ Trade form loaded');
    
    // Wait a bit for dropdown to populate
    await page.waitForTimeout(1000);
    const pairOptions = await page.locator('#trade-pair option').count();
    console.log('✅ Currency pair options:', pairOptions);
    
    // Debug: print all options
    if (pairOptions <= 1) {
      const html = await page.locator('#trade-pair').innerHTML();
      console.log('   Dropdown HTML:', html.substring(0, 200));
    }
    
    // 6. Test Execute Trade
    console.log('\n=== Testing: Execute Trade ===');
    // Select by text instead of index
    await page.selectOption('#trade-pair', { label: 'EUR/USD' });
    await page.selectOption('#trade-type', 'BUY');
    await page.fill('#trade-amount', '1000');
    await page.fill('#trade-price', '1.085');
    await page.click('#trade-form button[type="submit"]');
    
    // Wait for result
    await page.waitForTimeout(2000);
    const tradeResult = await page.locator('#trade-result').textContent();
    console.log('✅ Trade result:', tradeResult.substring(0, 100));
    
    // 7. Test Navigation - Positions
    console.log('\n=== Testing: Positions Page ===');
    await page.click('[data-view="positions"]');
    await page.waitForTimeout(1000);
    const positionsTable = await page.locator('#positions-table').isVisible();
    console.log('✅ Positions table visible:', positionsTable);
    
    // 8. Test Navigation - History
    console.log('\n=== Testing: History Page ===');
    await page.click('[data-view="history"]');
    await page.waitForTimeout(1000);
    const historyTable = await page.locator('#history-table').isVisible();
    console.log('✅ History table visible:', historyTable);
    
    const historyRows = await page.locator('#history-tbody tr').count();
    console.log('✅ History rows:', historyRows);
    
    // 9. Test Navigation - Predictions
    console.log('\n=== Testing: Predictions Page ===');
    await page.click('[data-view="predictions"]');
    await page.waitForSelector('#prediction-form', { timeout: 5000 });
    console.log('✅ Prediction form loaded');
    
    await page.waitForTimeout(1000);
    // Check if options are loaded
    const predOptions = await page.locator('#pred-pair option').count();
    console.log('✅ Prediction pair options:', predOptions);
    
    // Select EUR/USD explicitly by value
    await page.selectOption('#pred-pair', '1');
    await page.fill('#pred-periods', '10');
    await page.click('#prediction-form button[type="submit"]');
    
    await page.waitForTimeout(3000);
    const predResult = await page.locator('#prediction-result').textContent();
    console.log('✅ Prediction result:', predResult.substring(0, 150));
    
    // 10. Test Navigation - Backtest
    console.log('\n=== Testing: Backtest Page ===');
    await page.click('[data-view="backtest"]');
    await page.waitForSelector('#backtest-form', { timeout: 5000 });
    console.log('✅ Backtest form loaded');
    
    // 11. Test Logout
    console.log('\n=== Testing: Logout ===');
    await page.click('#logout-btn');
    await page.waitForSelector('#login-form', { timeout: 5000 });
    console.log('✅ Logout successful - Back to login');
    
    // Summary
    console.log('\n========================================');
    console.log('ALL TESTS COMPLETED SUCCESSFULLY!');
    console.log('========================================');
    
  } catch (error) {
    console.error('❌ Test failed:', error.message);
    
    // Take screenshot for debugging
    try {
      await page.screenshot({ path: '/tmp/test_failure.png' });
      console.log('Screenshot saved to /tmp/test_failure.png');
    } catch (e) {}
  } finally {
    await browser.close();
  }
})();
