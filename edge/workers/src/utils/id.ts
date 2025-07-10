const ALPHABET = '0123456789abcdefghijklmnopqrstuvwxyz'

export function generateId(prefix: string): string {
  const timestamp = Date.now().toString(36)
  const randomPart = Array.from({ length: 12 }, () => 
    ALPHABET[Math.floor(Math.random() * ALPHABET.length)]
  ).join('')
  
  return `${prefix}_${timestamp}${randomPart}`
}

export function generateShortCode(length = 6): string {
  return Array.from({ length }, () => 
    ALPHABET[Math.floor(Math.random() * ALPHABET.length)]
  ).join('').toUpperCase()
}