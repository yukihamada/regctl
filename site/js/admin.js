// Admin Dashboard JavaScript
// regctl Cloud 管理画面の機能実装

class AdminDashboard {
    constructor() {
        this.apiBaseUrl = 'https://api.regctl.com';
        this.refreshInterval = 5 * 60 * 1000; // 5分
        this.cache = new Map();
        this.checkAuth();
        this.init();
    }
    
    checkAuth() {
        const token = this.getAuthToken();
        const loginTime = localStorage.getItem('regctl_admin_login_time');
        
        if (!token || !loginTime) {
            this.redirectToLogin();
            return;
        }
        
        // 24時間でトークン期限切れ
        const isExpired = Date.now() - parseInt(loginTime) > 24 * 60 * 60 * 1000;
        
        if (isExpired) {
            this.logout();
            return;
        }
        
        // トークンの有効性をAPIで確認（バックグラウンド）
        this.validateToken().catch(() => {
            console.warn('Token validation failed, but continuing with cached token');
        });
    }
    
    async validateToken() {
        try {
            const response = await fetch(`${this.apiBaseUrl}/admin/auth/validate`, {
                headers: {
                    'Authorization': `Bearer ${this.getAuthToken()}`
                }
            });
            
            if (!response.ok) {
                throw new Error('Token validation failed');
            }
            
            return true;
        } catch (error) {
            console.error('Token validation error:', error);
            // 本番環境では logout() を呼ぶべき
            // this.logout();
            throw error;
        }
    }
    
    logout() {
        localStorage.removeItem('regctl_admin_token');
        localStorage.removeItem('regctl_admin_login_time');
        this.redirectToLogin();
    }
    
    redirectToLogin() {
        window.location.href = '/admin-login.html';
    }

    async init() {
        // 初期データ読み込み
        await this.loadAllData();
        
        // 定期更新設定
        setInterval(() => this.loadAllData(), this.refreshInterval);
        
        // 最終更新時刻更新
        this.updateLastRefreshTime();
        
        console.log('🛠️ Admin Dashboard initialized');
    }

    async loadAllData() {
        try {
            // 並列でデータ取得
            await Promise.allSettled([
                this.loadValueDomainData(),
                this.loadRoute53Data(),
                this.loadPorkbunData(),
                this.loadSystemStatus(),
                this.loadDomainList()
            ]);
            
            this.updateLastRefreshTime();
        } catch (error) {
            console.error('❌ Failed to load dashboard data:', error);
        }
    }

    async apiCall(endpoint, options = {}) {
        const cacheKey = `${endpoint}_${JSON.stringify(options)}`;
        
        // キャッシュチェック（5分以内）
        if (this.cache.has(cacheKey)) {
            const cached = this.cache.get(cacheKey);
            if (Date.now() - cached.timestamp < 5 * 60 * 1000) {
                return cached.data;
            }
        }

        try {
            const response = await fetch(`${this.apiBaseUrl}${endpoint}`, {
                headers: {
                    'Authorization': `Bearer ${this.getAuthToken()}`,
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`API Error: ${response.status} ${response.statusText}`);
            }

            const data = await response.json();
            
            // キャッシュに保存
            this.cache.set(cacheKey, {
                data,
                timestamp: Date.now()
            });

            return data;
        } catch (error) {
            console.error(`❌ API call failed for ${endpoint}:`, error);
            throw error;
        }
    }

    getAuthToken() {
        // TODO: 実際の認証トークン取得実装
        return localStorage.getItem('regctl_admin_token') || 'demo_token';
    }

    // ValueDomain データ読み込み
    async loadValueDomainData() {
        const startTime = Date.now();
        
        try {
            this.setElementStatus('vd-status', 'loading');
            
            // デモデータ（実際のAPI実装時に置き換え）
            const demoData = await this.simulateApiCall({
                balance: 125000,
                domains: 12,
                status: 'active',
                responseTime: Math.random() * 500 + 100
            });

            const responseTime = Date.now() - startTime;
            
            // UI更新
            document.getElementById('vd-balance').textContent = `¥${demoData.balance.toLocaleString()}`;
            document.getElementById('vd-response-time').textContent = `${responseTime}ms`;
            document.getElementById('vd-domain-count').textContent = `${demoData.domains}個`;
            
            this.setElementStatus('vd-status', 'online');
            this.hideError('vd-error');
            
        } catch (error) {
            this.setElementStatus('vd-status', 'offline');
            this.showError('vd-error', 'VALUE-DOMAIN API接続エラー: ' + error.message);
            
            // デフォルト値設定
            document.getElementById('vd-balance').textContent = '---';
            document.getElementById('vd-response-time').textContent = 'エラー';
            document.getElementById('vd-domain-count').textContent = '---';
        }
    }

    // Route53 データ読み込み
    async loadRoute53Data() {
        const startTime = Date.now();
        
        try {
            this.setElementStatus('r53-status', 'loading');
            
            // デモデータ
            const demoData = await this.simulateApiCall({
                monthlyCost: 45.23,
                hostedZones: 8,
                queries: 1234567,
                status: 'active'
            });

            // UI更新
            document.getElementById('r53-cost').textContent = `$${demoData.monthlyCost}`;
            document.getElementById('r53-zones').textContent = `${demoData.hostedZones}個`;
            document.getElementById('r53-queries').textContent = `${(demoData.queries / 1000000).toFixed(1)}M`;
            
            this.setElementStatus('r53-status', 'online');
            this.hideError('r53-error');
            
        } catch (error) {
            this.setElementStatus('r53-status', 'offline');
            this.showError('r53-error', 'Route53 API接続エラー: ' + error.message);
            
            document.getElementById('r53-cost').textContent = '---';
            document.getElementById('r53-zones').textContent = '---';
            document.getElementById('r53-queries').textContent = '---';
        }
    }

    // Porkbun データ読み込み
    async loadPorkbunData() {
        const startTime = Date.now();
        
        try {
            this.setElementStatus('pb-status', 'loading');
            
            // デモデータ
            const demoData = await this.simulateApiCall({
                balance: 89.45,
                domains: 5,
                status: 'active',
                responseTime: Math.random() * 300 + 150
            });

            const responseTime = Date.now() - startTime;

            // UI更新
            document.getElementById('pb-balance').textContent = `$${demoData.balance}`;
            document.getElementById('pb-response-time').textContent = `${responseTime}ms`;
            document.getElementById('pb-domain-count').textContent = `${demoData.domains}個`;
            
            this.setElementStatus('pb-status', 'online');
            this.hideError('pb-error');
            
        } catch (error) {
            this.setElementStatus('pb-status', 'offline');
            this.showError('pb-error', 'Porkbun API接続エラー: ' + error.message);
            
            document.getElementById('pb-balance').textContent = '---';
            document.getElementById('pb-response-time').textContent = 'エラー';
            document.getElementById('pb-domain-count').textContent = '---';
        }
    }

    // システム状態読み込み
    async loadSystemStatus() {
        try {
            this.setElementStatus('system-status', 'loading');
            
            // ヘルスチェック
            const healthChecks = await Promise.allSettled([
                this.checkWorkerStatus(),
                this.checkDatabaseStatus(),
                this.checkCacheStatus(),
                this.checkQueueStatus()
            ]);

            // 結果をUI反映
            document.getElementById('workers-status').textContent = 
                healthChecks[0].status === 'fulfilled' ? '🟢 稼働中' : '🔴 エラー';
            document.getElementById('db-status').textContent = 
                healthChecks[1].status === 'fulfilled' ? '🟢 稼働中' : '🔴 エラー';
            document.getElementById('cache-status').textContent = 
                healthChecks[2].status === 'fulfilled' ? '🟢 稼働中' : '🔴 エラー';
            document.getElementById('queue-status').textContent = 
                healthChecks[3].status === 'fulfilled' ? '🟢 稼働中' : '🔴 エラー';

            const allHealthy = healthChecks.every(check => check.status === 'fulfilled');
            this.setElementStatus('system-status', allHealthy ? 'online' : 'offline');
            
        } catch (error) {
            this.setElementStatus('system-status', 'offline');
            console.error('❌ System status check failed:', error);
        }
    }

    // ドメイン一覧読み込み
    async loadDomainList() {
        const domainList = document.getElementById('domain-list');
        
        try {
            // ローディング表示
            domainList.innerHTML = `
                <div style="text-align: center; padding: 2rem; color: var(--text-muted);">
                    <div style="animation: spin 1s linear infinite; display: inline-block;">🔄</div>
                    <div>ドメインデータを読み込み中...</div>
                </div>
            `;

            // デモデータ
            const domains = await this.simulateApiCall([
                { name: 'example.com', provider: 'VALUE-DOMAIN', status: 'active', expires: '2025-12-01' },
                { name: 'test.org', provider: 'Route53', status: 'active', expires: '2025-11-15' },
                { name: 'demo.net', provider: 'Porkbun', status: 'pending', expires: '2025-10-30' },
                { name: 'old.com', provider: 'VALUE-DOMAIN', status: 'expired', expires: '2024-08-15' },
                { name: 'regctl.com', provider: 'Cloudflare', status: 'active', expires: '2026-01-01' },
                { name: 'api.regctl.com', provider: 'Cloudflare', status: 'active', expires: '2026-01-01' },
                { name: 'app.regctl.com', provider: 'Cloudflare', status: 'active', expires: '2026-01-01' }
            ]);

            // ドメインリスト生成
            domainList.innerHTML = domains.map(domain => `
                <div class="domain-item" onclick="showDomainDetails('${domain.name}')">
                    <div>
                        <div class="domain-name">${domain.name}</div>
                        <div style="font-size: 0.8rem; color: var(--text-secondary);">
                            ${domain.provider} | 期限: ${domain.expires}
                        </div>
                    </div>
                    <div class="domain-status status-${domain.status}">
                        ${this.getStatusText(domain.status)}
                    </div>
                </div>
            `).join('');

        } catch (error) {
            domainList.innerHTML = `
                <div class="error-message">
                    ❌ ドメインリストの読み込みに失敗しました: ${error.message}
                </div>
            `;
        }
    }

    // ユーティリティメソッド
    async simulateApiCall(data, delay = 500) {
        // 開発中のデモデータ生成
        await new Promise(resolve => setTimeout(resolve, delay + Math.random() * 500));
        
        // 10%の確率でエラーをシミュレート
        if (Math.random() < 0.1) {
            throw new Error('Network timeout');
        }
        
        return data;
    }

    setElementStatus(elementId, status) {
        const element = document.getElementById(elementId);
        if (!element) return;

        element.className = 'status-indicator';
        
        switch (status) {
            case 'online':
                element.classList.add('status-online');
                break;
            case 'offline':
                element.classList.add('status-offline');
                break;
            case 'loading':
                element.style.background = 'var(--warning-color)';
                element.style.animation = 'pulse 1s infinite';
                break;
        }
    }

    showError(elementId, message) {
        const element = document.getElementById(elementId);
        if (!element) return;
        
        element.innerHTML = `<div class="error-message">${message}</div>`;
        element.style.display = 'block';
    }

    hideError(elementId) {
        const element = document.getElementById(elementId);
        if (!element) return;
        
        element.style.display = 'none';
    }

    getStatusText(status) {
        const statusMap = {
            'active': 'アクティブ',
            'pending': '処理中',
            'expired': '期限切れ',
            'suspended': '停止中'
        };
        return statusMap[status] || status;
    }

    updateLastRefreshTime() {
        const now = new Date();
        const timeString = now.toLocaleString('ja-JP', {
            year: 'numeric',
            month: '2-digit', 
            day: '2-digit',
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit'
        });
        
        document.getElementById('update-time').textContent = timeString;
    }

    // ヘルスチェック個別メソッド
    async checkWorkerStatus() {
        return this.apiCall('/health');
    }

    async checkDatabaseStatus() {
        return this.apiCall('/health/database');
    }

    async checkCacheStatus() {
        return this.apiCall('/health/cache');
    }

    async checkQueueStatus() {
        return this.apiCall('/health/queue');
    }
}

// グローバル関数（HTML から呼び出し）
let dashboard;

// リフレッシュ関数
async function refreshValueDomain() {
    const btn = document.getElementById('vd-refresh-text');
    const originalText = btn.textContent;
    
    btn.textContent = '⏳ 更新中...';
    try {
        await dashboard.loadValueDomainData();
    } finally {
        btn.textContent = originalText;
    }
}

async function refreshRoute53() {
    const btn = document.getElementById('r53-refresh-text');
    const originalText = btn.textContent;
    
    btn.textContent = '⏳ 更新中...';
    try {
        await dashboard.loadRoute53Data();
    } finally {
        btn.textContent = originalText;
    }
}

async function refreshPorkbun() {
    const btn = document.getElementById('pb-refresh-text');
    const originalText = btn.textContent;
    
    btn.textContent = '⏳ 更新中...';
    try {
        await dashboard.loadPorkbunData();
    } finally {
        btn.textContent = originalText;
    }
}

async function refreshSystemStatus() {
    const btn = document.getElementById('system-refresh-text');
    const originalText = btn.textContent;
    
    btn.textContent = '⏳ チェック中...';
    try {
        await dashboard.loadSystemStatus();
    } finally {
        btn.textContent = originalText;
    }
}

async function refreshDomains() {
    const btn = document.getElementById('domain-refresh-btn');
    btn.disabled = true;
    btn.textContent = '⏳ 更新中...';
    
    try {
        await dashboard.loadDomainList();
    } finally {
        btn.disabled = false;
        btn.textContent = '🔄 更新';
    }
}

// 詳細表示関数
function showValueDomainDetails() {
    alert('VALUE-DOMAIN 詳細画面を表示します（未実装）');
}

function showRoute53Details() {
    alert('Route53 詳細画面を表示します（未実装）');
}

function showPorkbunDetails() {
    alert('Porkbun 詳細画面を表示します（未実装）');
}

function showSystemLogs() {
    alert('システムログ画面を表示します（未実装）');
}

function showDomainDetails(domainName) {
    alert(`${domainName} の詳細画面を表示します（未実装）`);
}

// アクション関数
function addNewDomain() {
    alert('ドメイン追加画面を表示します（未実装）');
}

function exportDomains() {
    alert('ドメインリストをエクスポートします（未実装）');
}

function bulkActions() {
    alert('一括操作画面を表示します（未実装）');
}

// ログアウト関数
function logout() {
    if (confirm('ログアウトしますか？')) {
        dashboard.logout();
    }
}

// アニメーション用CSS
const style = document.createElement('style');
style.textContent = `
    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
    }
    
    @keyframes pulse {
        0%, 100% { opacity: 1; }
        50% { opacity: 0.5; }
    }
`;
document.head.appendChild(style);

// 初期化
document.addEventListener('DOMContentLoaded', () => {
    dashboard = new AdminDashboard();
    
    // エラーハンドリング
    window.addEventListener('unhandledrejection', event => {
        console.error('❌ Unhandled promise rejection:', event.reason);
    });
});

// モバイル対応
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js')
            .then(registration => {
                console.log('✅ SW registered:', registration);
            })
            .catch(registrationError => {
                console.log('❌ SW registration failed:', registrationError);
            });
    });
}