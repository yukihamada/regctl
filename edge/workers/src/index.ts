import { Hono } from 'hono'

const app = new Hono()

// 最小限のテスト
app.get('/', (c) => {
  return c.json({ message: 'regctl API is running', version: '0.1.0' })
})

app.get('/health', (c) => {
  return c.json({ status: 'healthy' })
})

// Durable Objects (必要最小限の実装)
export class RateLimiter {
  constructor(private state: any, private env: any) {}
  
  async fetch(request: Request): Promise<Response> {
    return new Response('OK')
  }
}

export default app