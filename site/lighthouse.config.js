module.exports = {
  ci: {
    collect: {
      staticDistDir: './',
      numberOfRuns: 3,
    },
    assert: {
      preset: 'lighthouse:no-pwa',
      assertions: {
        // パフォーマンス
        'first-contentful-paint': ['error', {maxNumericValue: 2000}],
        'largest-contentful-paint': ['error', {maxNumericValue: 2500}],
        'total-blocking-time': ['error', {maxNumericValue: 300}],
        'cumulative-layout-shift': ['error', {maxNumericValue: 0.1}],
        'speed-index': ['error', {maxNumericValue: 3000}],
        
        // アクセシビリティ
        'categories:accessibility': ['error', {minScore: 0.9}],
        
        // ベストプラクティス
        'categories:best-practices': ['error', {minScore: 0.9}],
        
        // SEO
        'categories:seo': ['error', {minScore: 0.9}],
        
        // 特定の監査項目
        'uses-webp-images': 'off', // 画像が少ないため
        'uses-http2': 'off', // Cloudflareが自動的に処理
        'uses-text-compression': 'off', // Cloudflareが自動的に処理
        
        // JavaScript関連
        'unused-javascript': ['warn', {maxNumericValue: 0}],
        'uses-long-cache-ttl': 'off', // Cloudflareが処理
        
        // その他の重要な指標
        'dom-size': ['error', {maxNumericValue: 1500}],
        'mainthread-work-breakdown': ['error', {maxNumericValue: 2000}],
        'bootup-time': ['error', {maxNumericValue: 2000}],
      },
    },
    upload: {
      target: 'temporary-public-storage',
    },
  },
};