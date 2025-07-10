import { Context } from 'hono'
import { HTTPException } from 'hono/http-exception'

export const errorHandler = (err: Error, c: Context) => {
  console.error('Error:', err)

  if (err instanceof HTTPException) {
    return c.json({
      error: err.message,
      status: err.status,
    }, err.status)
  }

  // Handle validation errors
  if (err.name === 'ZodError') {
    return c.json({
      error: 'Validation error',
      details: JSON.parse(err.message),
    }, 400)
  }

  // Default error response
  return c.json({
    error: 'Internal server error',
    message: c.env?.ENVIRONMENT === 'development' ? err.message : undefined,
  }, 500)
}