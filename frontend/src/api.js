const API_BASE = import.meta.env.VITE_API_BASE || ''

async function request(path) {
  const response = await fetch(`${API_BASE}${path}`)
  const payload = await response.json().catch(() => ({}))
  if (!response.ok) {
    throw new Error(payload.error || `HTTP ${response.status}`)
  }
  return payload
}

export function fetchClassrooms(campusId, slots) {
  const params = new URLSearchParams({ campusId: String(campusId) })
  if (slots && slots.length > 0) {
    params.set('slots', slots.join(','))
  }
  return request(`/api/classrooms?${params.toString()}`)
}

export function fetchSlots() {
  return request('/api/slots')
}
