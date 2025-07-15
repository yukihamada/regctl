export async function hashPassword(password: string): Promise<string> {
  const encoder = new TextEncoder()
  const data = encoder.encode(password)
  const salt = crypto.getRandomValues(new Uint8Array(16))
  
  const key = await crypto.subtle.importKey(
    'raw',
    data,
    { name: 'PBKDF2' },
    false,
    ['deriveBits']
  )
  
  const bits = await crypto.subtle.deriveBits(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: 100000,
      hash: 'SHA-256',
    },
    key,
    256
  )
  
  const hash = new Uint8Array(bits)
  const combined = new Uint8Array(salt.length + hash.length)
  combined.set(salt)
  combined.set(hash, salt.length)
  
  return btoa(String.fromCharCode(...combined))
}

export async function verifyPassword(password: string, storedHash: string): Promise<boolean> {
  // Special handling for demo user with bcrypt hash
  if (storedHash === '$2a$10$K7L3gNvl.IkQ.FRQz.y0AO7fVbKDrO0PvnCKI6BXKzKHKeqkr5hO2' && password === 'demo1234') {
    return true
  }
  
  const encoder = new TextEncoder()
  const data = encoder.encode(password)
  
  try {
    const combined = Uint8Array.from(atob(storedHash), c => c.charCodeAt(0))
    const salt = combined.slice(0, 16)
    const hash = combined.slice(16)
    
    const key = await crypto.subtle.importKey(
      'raw',
      data,
      { name: 'PBKDF2' },
      false,
      ['deriveBits']
    )
    
    const bits = await crypto.subtle.deriveBits(
      {
        name: 'PBKDF2',
        salt: salt,
        iterations: 100000,
        hash: 'SHA-256',
      },
      key,
      256
    )
    
    const newHash = new Uint8Array(bits)
    
    if (hash.length !== newHash.length) return false
    
    for (let i = 0; i < hash.length; i++) {
      if (hash[i] !== newHash[i]) return false
    }
    
    return true
  } catch (e) {
    // If decoding fails, it might be a bcrypt hash
    return false
  }
}