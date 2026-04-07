import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export interface User {
  id: number
  username: string
  role: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<User | null>(JSON.parse(localStorage.getItem('user') || 'null'))
  const adminToken = ref<string | null>(localStorage.getItem('adminToken'))
  const adminUser = ref<User | null>(JSON.parse(localStorage.getItem('adminUser') || 'null'))

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isImpersonating = computed(() => !!adminToken.value)

  async function login(username: string, password: string) {
    const res = await fetch('/api/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    if (!res.ok) {
      const data = await res.json()
      throw new Error(data.error || 'Login failed')
    }
    const data = await res.json()
    token.value = data.token
    user.value = data.user
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
  }

  async function impersonate(userId: number) {
    adminToken.value = token.value
    adminUser.value = user.value
    localStorage.setItem('adminToken', token.value!)
    localStorage.setItem('adminUser', JSON.stringify(user.value))

    const res = await fetch(`/api/admin/impersonate/${userId}`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${adminToken.value}` },
    })
    if (!res.ok) {
      // Restore on failure
      adminToken.value = null
      adminUser.value = null
      localStorage.removeItem('adminToken')
      localStorage.removeItem('adminUser')
      const data = await res.json()
      throw new Error(data.error || 'Impersonation failed')
    }
    const data = await res.json()
    token.value = data.token
    user.value = data.user
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(data.user))
  }

  function stopImpersonating() {
    token.value = adminToken.value
    user.value = adminUser.value
    localStorage.setItem('token', adminToken.value!)
    localStorage.setItem('user', JSON.stringify(adminUser.value))
    adminToken.value = null
    adminUser.value = null
    localStorage.removeItem('adminToken')
    localStorage.removeItem('adminUser')
  }

  function logout() {
    token.value = null
    user.value = null
    adminToken.value = null
    adminUser.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    localStorage.removeItem('adminToken')
    localStorage.removeItem('adminUser')
  }

  return { token, user, adminUser, isLoggedIn, isAdmin, isImpersonating, login, impersonate, stopImpersonating, logout }
})
