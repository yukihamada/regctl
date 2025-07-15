/**
 * regctl.com A/B Testing Worker
 * 
 * このWorkerはエッジレベルでA/Bテストを実行し、
 * 高速で低レイテンシなテスト体験を提供します。
 */

interface Env {
  ENVIRONMENT: string;
  AB_TEST_HERO_CTA: string;
  AB_TEST_PRICING_ORDER: string;
  AB_TEST_DEMO_DEFAULT: string;
}

// A/Bテスト設定
const AB_TESTS = {
  heroCTA: {
    variants: [
      "⚡ 今すぐ始める",      // Control
      "🚀 3秒で始める",      // Variant A
      "💡 無料で試す",       // Variant B
      "🤖 Claude Code連携"   // Variant C
    ]
  },
  pricingOrder: {
    variants: [
      "standard-first",  // Control: スタンダード最初
      "starter-first",   // Variant A: スターター最初
      "pro-first"        // Variant B: プロ最初
    ]
  },
  demoDefault: {
    variants: [
      "cli",         // Control: CLI
      "claude",      // Variant A: Claude Code
      "api"          // Variant B: JSON API
    ]
  }
};

// ユーザーをテストバリアントに割り当て
function assignUserToVariant(userId: string, testName: string): number {
  const hash = simpleHash(`${userId}-${testName}`);
  const variantCount = AB_TESTS[testName as keyof typeof AB_TESTS].variants.length;
  return hash % variantCount;
}

// シンプルなハッシュ関数
function simpleHash(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // 32bit整数に変換
  }
  return Math.abs(hash);
}

// HTMLを変更してA/Bテストバリアントを適用
function applyABTestVariants(html: string, variants: Record<string, number>, userId: string): string {
  let modifiedHtml = html;

  // Hero CTA テスト
  if (variants.heroCTA !== undefined) {
    const ctaText = AB_TESTS.heroCTA.variants[variants.heroCTA];
    modifiedHtml = modifiedHtml.replace(
      /<span>⚡ 今すぐ始める<\/span>/g,
      `<span>${ctaText}</span>`
    );
  }

  // Pricing Order テスト
  if (variants.pricingOrder !== undefined) {
    const orderType = AB_TESTS.pricingOrder.variants[variants.pricingOrder];
    modifiedHtml = modifiedHtml.replace(
      '<div class="pricing-grid">',
      `<div class="pricing-grid" data-order="${orderType}">`
    );
  }

  // Demo Default テスト  
  if (variants.demoDefault !== undefined) {
    const defaultTab = AB_TESTS.demoDefault.variants[variants.demoDefault];
    modifiedHtml = modifiedHtml.replace(
      'showDemo(\'cli\')',
      `showDemo('${defaultTab}')`
    );
  }

  // A/Bテストの識別用のメタタグを追加
  const testInfo = Object.entries(variants)
    .map(([test, variant]) => `${test}:${variant}`)
    .join(',');
  
  modifiedHtml = modifiedHtml.replace(
    '<head>',
    `<head>\n    <meta name="ab-test-variants" content="${testInfo}" data-user="${userId}">`
  );

  return modifiedHtml;
}

// アナリティクス用のイベント送信
function sendAnalyticsEvent(userId: string, event: string, variants: Record<string, number>) {
  // Google Analytics 4 または Cloudflare Analytics に送信
  // 実装時には適切なアナリティクスサービスのAPIを使用
  console.log(`Analytics: User ${userId}, Event: ${event}, Variants:`, variants);
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    
    // A/Bテストが無効な場合は直接パススルー
    if (env.ENVIRONMENT === 'development') {
      return fetch(request);
    }

    // ユーザーIDを取得または生成
    let userId = request.headers.get('cf-connecting-ip') || 'anonymous';
    const userAgent = request.headers.get('user-agent') || '';
    userId = simpleHash(userId + userAgent).toString();

    // A/Bテストバリアントを決定
    const variants: Record<string, number> = {};
    
    if (env.AB_TEST_HERO_CTA === 'true') {
      variants.heroCTA = assignUserToVariant(userId, 'heroCTA');
    }
    
    if (env.AB_TEST_PRICING_ORDER === 'true') {
      variants.pricingOrder = assignUserToVariant(userId, 'pricingOrder');
    }
    
    if (env.AB_TEST_DEMO_DEFAULT === 'true') {
      variants.demoDefault = assignUserToVariant(userId, 'demoDefault');
    }

    // オリジナルのレスポンスを取得
    const upstreamURL = `https://regctl-site.pages.dev${url.pathname}${url.search}`;
    const upstreamRequest = new Request(upstreamURL, {
      method: request.method,
      headers: request.headers,
      body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : null
    });
    
    const response = await fetch(upstreamRequest);

    // HTMLページの場合のみA/Bテストを適用
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('text/html')) {
      const html = await response.text();
      const modifiedHtml = applyABTestVariants(html, variants, userId);
      
      // A/Bテスト参加イベントを送信
      sendAnalyticsEvent(userId, 'ab_test_view', variants);
      
      const newHeaders = new Headers(response.headers);
      newHeaders.set('x-ab-test-user', userId);
      newHeaders.set('x-ab-test-variants', Object.entries(variants)
        .map(([test, variant]) => `${test}:${variant}`)
        .join(','));
      newHeaders.set('content-type', 'text/html; charset=utf-8');
      
      return new Response(modifiedHtml, {
        status: response.status,
        statusText: response.statusText,
        headers: newHeaders
      });
    }

    // HTMLでない場合はそのまま返す
    return response;
  }
};