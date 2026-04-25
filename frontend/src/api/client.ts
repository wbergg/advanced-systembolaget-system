// Auth-aware fetch wrapper
async function authFetch(url: string, init?: RequestInit): Promise<Response> {
  const token = localStorage.getItem('token')
  const headers = new Headers(init?.headers)
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const res = await fetch(url, { ...init, headers })
  if (res.status === 401) {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    localStorage.removeItem('adminToken')
    localStorage.removeItem('adminUser')
    window.location.reload()
  }
  return res
}

// Auth API

export interface AuthUser {
  id: number
  username: string
  role: string
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  const res = await authFetch('/api/me/password', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || 'Failed to change password')
  }
}

// Admin API

export async function listUsers(): Promise<AuthUser[]> {
  const res = await authFetch('/api/admin/users')
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function createUser(username: string, password: string, role: string): Promise<AuthUser> {
  const res = await authFetch('/api/admin/users', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password, role }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || 'Failed to create user')
  }
  return res.json()
}

export async function updateUser(id: number, data: { username?: string; password?: string; role?: string }): Promise<void> {
  const res = await authFetch(`/api/admin/users/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const d = await res.json()
    throw new Error(d.error || 'Failed to update user')
  }
}

export async function deleteUser(id: number): Promise<void> {
  const res = await authFetch(`/api/admin/users/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

export async function deleteProduct(id: string): Promise<void> {
  const res = await authFetch(`/api/admin/products/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

export async function deleteAllProducts(): Promise<{ deleted: number }> {
  const res = await authFetch('/api/admin/products', { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function debugSBProbe(
  number: string,
  filters: Record<string, string> = {},
): Promise<any> {
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(filters)) {
    if (v !== undefined && v !== null && v !== '') params.set(k, v)
  }
  const qs = params.toString()
  const url = `/api/admin/debug/sb-probe/${encodeURIComponent(number)}${qs ? '?' + qs : ''}`
  const res = await authFetch(url)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

// Products

export interface Product {
  productId: string
  productNumber: string
  productNameBold: string
  productNameThin: string | null
  producerName: string
  price: number
  volume: number
  volumeText: string
  alcoholPercentage: number
  country: string
  categoryLevel1: string
  categoryLevel2: string
  assortmentText: string
  taste: string
  usage: string
  isOrganic: boolean
  isNews: boolean
  packagingLevel1: string
  assortment: string
  restrictedParcelQuantity: number
  vintage: string | null
  imageUrl: string
  note: string | null
}

export interface ProductsResponse {
  products: Product[]
  total: number
  page: number
  pageSize: number
}

export interface ListParams {
  search?: string
  category?: string
  minPrice?: number
  maxPrice?: number
  minAbv?: number
  maxAbv?: number
  sortBy?: string
  sortDir?: string
  page?: number
  pageSize?: number
  name?: string
  producer?: string
  country?: string[]
  packaging?: string[]
  volume?: string[]
}

export async function getProducts(params: ListParams): Promise<ProductsResponse> {
  const q = new URLSearchParams()
  if (params.search) q.set('search', params.search)
  if (params.category) q.set('category', params.category)
  if (params.minPrice != null) q.set('minPrice', String(params.minPrice))
  if (params.maxPrice != null) q.set('maxPrice', String(params.maxPrice))
  if (params.minAbv != null) q.set('minAbv', String(params.minAbv))
  if (params.maxAbv != null) q.set('maxAbv', String(params.maxAbv))
  if (params.sortBy) q.set('sortBy', params.sortBy)
  if (params.sortDir) q.set('sortDir', params.sortDir)
  if (params.page) q.set('page', String(params.page))
  if (params.pageSize) q.set('pageSize', String(params.pageSize))
  if (params.name) q.set('name', params.name)
  if (params.producer) q.set('producer', params.producer)
  if (params.country && params.country.length > 0) q.set('country', params.country.join(','))
  if (params.packaging && params.packaging.length > 0) q.set('packaging', params.packaging.join(','))
  if (params.volume && params.volume.length > 0) q.set('volume', params.volume.join(','))

  const res = await authFetch(`/api/products?${q}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getDistinctValues(column: string): Promise<string[]> {
  const res = await authFetch(`/api/products/distinct/${column}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getProduct(id: string): Promise<Product> {
  const res = await authFetch(`/api/products/${id}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getProductByNumber(number: string): Promise<Product> {
  const res = await authFetch(`/api/products/by-number/${number}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export interface SyncProgress {
  page: number
  totalPages: number
  products: number
}

export async function syncProducts(
  filters: Record<string, string>,
  onProgress?: (p: SyncProgress) => void,
): Promise<{ synced: number }> {
  const res = await authFetch('/api/sync', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ filters }),
  })
  if (!res.ok) throw new Error(await res.text())

  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  let result: { synced: number } | null = null

  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })

    const lines = buf.split('\n')
    buf = lines.pop()!

    let event = ''
    for (const line of lines) {
      if (line.startsWith('event:')) {
        event = line.slice(6).trim()
      } else if (line.startsWith('data:')) {
        const data = JSON.parse(line.slice(5).trim())
        if (event === 'progress' && onProgress) {
          onProgress(data)
        } else if (event === 'error') {
          throw new Error(data.error)
        } else if (event === 'done') {
          result = data
        }
      }
    }
  }

  if (!result) throw new Error('Sync stream ended without result')
  return result
}

export async function refreshKey(): Promise<void> {
  const res = await authFetch('/api/key/refresh', { method: 'POST' })
  if (!res.ok) throw new Error(await res.text())
}

export async function getKeyStatus(): Promise<{ hasKey: boolean }> {
  const res = await authFetch('/api/key/status')
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function updateNote(id: string, note: string): Promise<void> {
  const res = await authFetch(`/api/products/${id}/notes`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note }),
  })
  if (!res.ok) throw new Error(await res.text())
}

// Comments

export interface Comment {
  id: number
  productId: string
  userId: number
  username: string
  comment: string
  createdAt: string
}

export async function getComments(productId: string): Promise<Comment[]> {
  const res = await authFetch(`/api/products/${productId}/comments`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function addComment(productId: string, comment: string): Promise<Comment> {
  const res = await authFetch(`/api/products/${productId}/comments`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ comment }),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function deleteComment(id: number): Promise<void> {
  const res = await authFetch(`/api/admin/comments/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

// Users

export interface ShareUser {
  userId: number
  username: string
}

export async function listAllUsers(): Promise<ShareUser[]> {
  const res = await authFetch('/api/users/list')
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

// Events

export interface EventAttendee {
  userId: number
  username: string
}

export interface EventBeer {
  id: number
  productId: string
  productNameBold: string
  productNameThin: string | null
  producerName: string
  imageUrl: string
}

export interface EventScore {
  eventBeerId: number
  userId: number
  score: number
}

export interface Event {
  id: number
  name: string
  description: string
  eventDate: string
  ownerId: number
  ownerName: string
  locked: boolean
  type: 'tasting' | 'roll'
  hidden: boolean
  public?: boolean
  createdAt: string
  attendees?: EventAttendee[]
  beers?: EventBeer[]
  scores?: EventScore[]
  attendeeCount: number
  beerCount: number
}

export async function listEvents(): Promise<Event[]> {
  const res = await authFetch('/api/events')
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function createEvent(name: string, description: string, eventDate: string, type: string = 'tasting'): Promise<Event> {
  const res = await authFetch('/api/events', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, eventDate, type }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to create event') }
  return res.json()
}

export async function getEvent(id: number): Promise<Event> {
  const res = await authFetch(`/api/events/${id}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function updateEvent(id: number, name: string, description: string, eventDate: string): Promise<void> {
  const res = await authFetch(`/api/events/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, description, eventDate }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to update event') }
}

export async function deleteEvent(id: number): Promise<void> {
  const res = await authFetch(`/api/events/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

export async function setEventLocked(eventId: number, locked: boolean): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/lock`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ locked }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to update lock') }
}

export async function inviteToEvent(eventId: number, userId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/invite`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to invite') }
}

export async function uninviteFromEvent(eventId: number, userId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/invite/${userId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

export async function importSharedListToEvent(eventId: number, listId: number, replace = false): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/import-list`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ listId, replace }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to import list') }
}

export async function addBeerToEvent(eventId: number, productId: string): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/beers`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ productId }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to add beer') }
}

export async function removeBeerFromEvent(eventId: number, beerId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/beers/${beerId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

export async function setScore(eventId: number, beerId: number, score: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/scores/${beerId}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ score }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to set score') }
}

export async function deleteScore(eventId: number, beerId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/scores/${beerId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

// Roll game

export interface RollPoolItem {
  id: number
  productId: string
  productNameBold: string
  productNameThin: string | null
  producerName: string
  imageUrl: string
  consumed: boolean
  consumedByUserId?: number
  consumedByName?: string
  consumedAt?: string
  vetoed: boolean
}

export interface RollTurn {
  id: number
  eventId: number
  poolId: number
  userId: number
  username: string
  productNameBold: string
  productNameThin: string | null
  producerName: string
  imageUrl: string
  status: 'pending' | 'accepted' | 'vetoed'
  canVeto: boolean
  createdAt: string
  resolvedAt?: string
}

export interface VetoedItem {
  poolId: number
  productNameBold: string
  productNameThin: string | null
  producerName: string
  imageUrl: string
  vetoedByName: string
  vetoedAt: string
}

export interface RollState {
  poolCount: number
  totalCount: number
  consumed: RollPoolItem[]
  vetoed: VetoedItem[]
  pendingTurn: RollTurn | null
  userVetoes: Record<number, boolean>
  finished: boolean
}

export async function getRollState(eventId: number): Promise<RollState> {
  const res = await authFetch(`/api/events/${eventId}/roll`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function performRoll(eventId: number, userId: number): Promise<RollTurn> {
  const res = await authFetch(`/api/events/${eventId}/roll`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to roll') }
  return res.json()
}

export async function acceptRoll(eventId: number, turnId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/roll/${turnId}/accept`, { method: 'POST' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to accept') }
}

export async function vetoRoll(eventId: number, turnId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/roll/${turnId}/veto`, { method: 'POST' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to veto') }
}

export async function resetRoll(eventId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/roll/reset`, { method: 'POST' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to reset') }
}

export async function undoVeto(eventId: number, poolId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/roll/veto/${poolId}`, { method: 'DELETE' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to undo veto') }
}

export async function undoConsumed(eventId: number, poolId: number): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/roll/pool/${poolId}`, { method: 'DELETE' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to undo') }
}

// Shared Lists

export interface SharedListItem {
  productId: string
  productNumber: string
  productNameBold: string
  productNameThin: string | null
  producerName: string
  price: number
  volume: number
  volumeText: string
  alcoholPercentage: number
  country: string
  categoryLevel1: string
  categoryLevel2: string
  packagingLevel1: string
  imageUrl: string
  taste: string
  usage: string
  isOrganic: boolean
  quantity: number
  addedBy: string
  addedAt: string
}

export interface SharedList {
  id: number
  uuid: string
  name: string
  userId: number
  ownerName: string
  shared: boolean
  locked: boolean
  sharedWith?: ShareUser[]
  itemCount: number
  total: number
  createdAt: string
  items?: SharedListItem[]
}

export async function listSharedLists(): Promise<SharedList[]> {
  const res = await authFetch('/api/shared-lists')
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function createSharedList(name: string): Promise<SharedList> {
  const res = await authFetch('/api/shared-lists', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getSharedList(id: number): Promise<SharedList> {
  const res = await authFetch(`/api/shared-lists/${id}`)
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function deleteSharedList(id: number): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

export async function addToSharedList(listId: number, productId: string, quantity = 1): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${listId}/items`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ productId, quantity }),
  })
  if (!res.ok) throw new Error(await res.text())
}

export async function removeFromSharedList(listId: number, productId: string): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${listId}/items/${productId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

export async function updateSharedListItemQty(listId: number, productId: string, quantity: number): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${listId}/items/${productId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ quantity }),
  })
  if (!res.ok) {
    const d = await res.json()
    throw new Error(d.error || 'Failed to update quantity')
  }
}

export async function renameSharedList(id: number, name: string): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  if (!res.ok) throw new Error(await res.text())
}

export async function setSharedListLocked(listId: number, locked: boolean): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${listId}/lock`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ locked }),
  })
  if (!res.ok) {
    const d = await res.json()
    throw new Error(d.error || 'Failed to update lock')
  }
}

export async function shareSharedList(listId: number, userId: number): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${listId}/share`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  })
  if (!res.ok) {
    const data = await res.json()
    throw new Error(data.error || 'Failed to share list')
  }
}

export async function unshareSharedList(listId: number, userId: number): Promise<void> {
  const res = await authFetch(`/api/shared-lists/${listId}/share/${userId}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(await res.text())
}

// Public (no auth) - for shared list page
export async function getPublicSharedList(uuid: string): Promise<SharedList> {
  const res = await fetch(`/api/public/shared-list/${uuid}`)
  if (!res.ok) throw new Error('List not found')
  return res.json()
}

export async function setEventHidden(eventId: number, hidden: boolean): Promise<void> {
  const res = await authFetch(`/api/events/${eventId}/visibility`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ hidden }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to update visibility') }
}

export async function toggleEventPublic(eventId: number): Promise<{ public: boolean }> {
  const res = await authFetch(`/api/events/${eventId}/public`, { method: 'PATCH' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to toggle public') }
  return res.json()
}

// Public roll endpoints (no auth)
export interface PublicRollData {
  eventName: string
  state: RollState
  participants: { userId: number; username: string }[]
}

export async function getPublicRoll(): Promise<PublicRollData> {
  const res = await fetch('/api/public/roll')
  if (!res.ok) throw new Error('Roll not found')
  return res.json()
}

export async function publicPerformRoll(userId: number): Promise<RollTurn> {
  const res = await fetch('/api/public/roll', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ userId }),
  })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to roll') }
  return res.json()
}

export async function publicAcceptRoll(turnId: number): Promise<void> {
  const res = await fetch(`/api/public/roll/${turnId}/accept`, { method: 'POST' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to accept') }
}

export async function publicVetoRoll(turnId: number): Promise<void> {
  const res = await fetch(`/api/public/roll/${turnId}/veto`, { method: 'POST' })
  if (!res.ok) { const d = await res.json(); throw new Error(d.error || 'Failed to veto') }
}
