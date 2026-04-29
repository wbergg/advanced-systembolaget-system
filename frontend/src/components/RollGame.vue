<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import Button from 'primevue/button'
import Select from 'primevue/select'
import {
  getRollState, performRoll, acceptRoll, vetoRoll, resetRoll, undoConsumed, undoVeto,
  getPublicRoll, publicPerformRoll, publicAcceptRoll, publicVetoRoll,
  type RollState,
} from '../api/client'
import { useAuthStore } from '../stores/auth'

const props = defineProps<{
  eventId: number
  participants: { userId: number; username: string }[]
  isPublic?: boolean
  canEdit?: boolean
}>()

const authStore = useAuthStore()
const state = ref<RollState | null>(null)
const rolling = ref(false)
const acting = ref(false)
const selectedUser = ref<{ userId: number; username: string } | null>(null)
const error = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

async function loadState() {
  try {
    if (props.isPublic) {
      const data = await getPublicRoll()
      state.value = data.state
    } else {
      state.value = await getRollState(props.eventId)
    }
    error.value = ''
  } catch (e: any) {
    error.value = e.message
  }
}

async function doRoll() {
  if (!selectedUser.value || rolling.value) return
  rolling.value = true
  error.value = ''
  try {
    if (props.isPublic) {
      await publicPerformRoll(selectedUser.value.userId)
    } else {
      await performRoll(props.eventId, selectedUser.value.userId)
    }
    await loadState()
  } catch (e: any) {
    error.value = e.message
  } finally {
    rolling.value = false
  }
}

async function doAccept() {
  if (!state.value?.pendingTurn || acting.value) return
  acting.value = true
  error.value = ''
  try {
    if (props.isPublic) {
      await publicAcceptRoll(state.value.pendingTurn.id)
    } else {
      await acceptRoll(props.eventId, state.value.pendingTurn.id)
    }
    await loadState()
  } catch (e: any) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function doVeto() {
  if (!state.value?.pendingTurn || acting.value) return
  acting.value = true
  error.value = ''
  try {
    if (props.isPublic) {
      await publicVetoRoll(state.value.pendingTurn.id)
    } else {
      await vetoRoll(props.eventId, state.value.pendingTurn.id)
    }
    await loadState()
  } catch (e: any) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function doUndoVeto(poolId: number) {
  if (acting.value) return
  acting.value = true
  error.value = ''
  try {
    await undoVeto(props.eventId, poolId)
    await loadState()
  } catch (e: any) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function doUndo(poolId: number) {
  if (acting.value) return
  acting.value = true
  error.value = ''
  try {
    await undoConsumed(props.eventId, poolId)
    await loadState()
  } catch (e: any) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function doReset() {
  if (acting.value) return
  acting.value = true
  error.value = ''
  try {
    await resetRoll(props.eventId)
    await loadState()
  } catch (e: any) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

const isAdmin = computed(() => !props.isPublic && authStore.user?.role === 'admin')
const canManage = computed(() => !props.isPublic && (isAdmin.value || !!props.canEdit))
const progressPct = computed(() => {
  if (!state.value || state.value.totalCount === 0) return 0
  return Math.round(((state.value.totalCount - state.value.poolCount) / state.value.totalCount) * 100)
})

function formatDate(dt: string | undefined) {
  if (!dt) return ''
  const d = new Date(dt)
  return d.toLocaleString('sv-SE', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

onMounted(() => {
  loadState()
  pollTimer = setInterval(loadState, 4000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div v-if="state" class="roll-game">
    <!-- Progress bar -->
    <div class="progress-section">
      <div class="progress-text">
        <span>{{ state.totalCount - state.poolCount }} of {{ state.totalCount }} beers consumed</span>
        <span class="progress-pct">{{ progressPct }}%</span>
      </div>
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progressPct + '%' }"></div>
      </div>
    </div>

    <!-- Game finished banner -->
    <div v-if="state.finished" class="finished-banner">
      All beers have been consumed! Game over.
    </div>

    <!-- Error -->
    <div v-if="error" class="error-banner">{{ error }}</div>

    <!-- Roll area -->
    <div v-if="!state.finished" class="roll-area">
      <!-- No pending turn: show roll controls -->
      <div v-if="!state.pendingTurn" class="roll-controls">
        <div class="roll-row">
          <Select
            v-model="selectedUser"
            :options="participants"
            optionLabel="username"
            placeholder="Select player..."
            style="flex: 1; min-width: 160px;"
          />
          <Button
            label="Roll"
            icon="pi pi-sync"
            @click="doRoll"
            :disabled="!selectedUser || rolling"
            :loading="rolling"
          />
        </div>
        <div class="roll-hint">{{ state.poolCount }} beers remaining in the pool</div>
      </div>

      <!-- Pending turn: show rolled beer -->
      <div v-else class="pending-turn">
        <div class="turn-header">
          <span class="turn-user">{{ state.pendingTurn.username }}</span> rolled:
        </div>
        <div class="beer-card">
          <img
            v-if="state.pendingTurn.imageUrl"
            :src="state.pendingTurn.imageUrl.replace('_400.', '_100.')"
            class="beer-img"
          />
          <div class="beer-info">
            <div class="beer-title">{{ state.pendingTurn.productNameBold }}</div>
            <div v-if="state.pendingTurn.productNameThin" class="beer-subtitle">
              {{ state.pendingTurn.productNameThin }}
            </div>
            <div class="beer-producer">
              {{ state.pendingTurn.producerName }}<span v-if="state.pendingTurn.alcoholPercent"> · {{ state.pendingTurn.alcoholPercent }}%</span>
            </div>
          </div>
        </div>

        <!-- Veto info -->
        <div v-if="!state.pendingTurn.canVeto" class="veto-info">
          <template v-if="state.userVetoes[state.pendingTurn.userId]">
            Veto already used — must accept.
          </template>
          <template v-else>
            This beer was previously vetoed — must accept.
          </template>
        </div>

        <!-- Action buttons -->
        <div class="turn-actions">
          <Button
            label="Accept"
            icon="pi pi-check"
            severity="success"
            @click="doAccept"
            :disabled="acting"
            :loading="acting"
          />
          <Button
            v-if="state.pendingTurn.canVeto"
            label="Veto"
            icon="pi pi-times"
            severity="danger"
            @click="doVeto"
            :disabled="acting"
          />
        </div>
      </div>
    </div>

    <!-- Host/admin reset -->
    <div v-if="canManage" class="admin-section">
      <Button label="Reset Game" icon="pi pi-refresh" severity="secondary" size="small" @click="doReset" :disabled="acting" />
    </div>

    <!-- Vetoed list -->
    <div v-if="state.vetoed.length > 0" class="vetoed-section">
      <div class="section-label">
        <i class="pi pi-ban" style="font-size: 0.75rem"></i> Vetoed
      </div>
      <div class="vetoed-list">
        <div v-for="(item, idx) in state.vetoed" :key="idx" class="vetoed-item">
          <img
            v-if="item.imageUrl"
            :src="item.imageUrl.replace('_400.', '_60.')"
            class="consumed-thumb"
          />
          <div class="consumed-info">
            <span class="consumed-name">{{ item.productNameBold }}</span>
            <span class="consumed-meta">
              <span v-if="item.alcoholPercent">{{ item.alcoholPercent }}% · </span>vetoed by <strong>{{ item.vetoedByName }}</strong> — {{ formatDate(item.vetoedAt) }}
            </span>
          </div>
          <i v-if="canManage" class="pi pi-undo consumed-undo" title="Undo veto — restore veto allowance" @click="doUndoVeto(item.poolId)"></i>
        </div>
      </div>
    </div>

    <!-- Consumed list -->
    <div v-if="state.consumed.length > 0" class="consumed-section">
      <div class="section-label">
        <i class="pi pi-check-circle" style="font-size: 0.75rem"></i> Consumed
      </div>
      <div class="consumed-list">
        <div v-for="item in state.consumed" :key="item.id" class="consumed-item">
          <img
            v-if="item.imageUrl"
            :src="item.imageUrl.replace('_400.', '_60.')"
            class="consumed-thumb"
          />
          <div class="consumed-info">
            <span class="consumed-name">{{ item.productNameBold }}</span>
            <span class="consumed-meta">
              <span v-if="item.alcoholPercent">{{ item.alcoholPercent }}% · </span>consumed by <strong>{{ item.consumedByName }}</strong> — {{ formatDate(item.consumedAt) }}
            </span>
          </div>
          <i v-if="canManage" class="pi pi-undo consumed-undo" title="Undo — put back in pool" @click="doUndo(item.id)"></i>
        </div>
      </div>
    </div>
  </div>

  <div v-else class="empty-state" style="padding: 2rem;">Loading game...</div>
</template>

<style scoped>
.roll-game {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.progress-section {
  padding: 0.75rem 1rem;
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.progress-text {
  display: flex;
  justify-content: space-between;
  font-size: 0.8rem;
  color: var(--text-muted);
  margin-bottom: 0.4rem;
}

.progress-pct {
  font-weight: 600;
}

.progress-bar {
  height: 6px;
  background: var(--border);
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 3px;
  transition: width 0.3s;
}

.finished-banner {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: var(--success-light);
  color: #065f46;
  border: 1px solid #6ee7b7;
  border-radius: var(--radius);
  padding: 1rem;
  font-weight: 600;
  font-size: 0.95rem;
  justify-content: center;
}

.error-banner {
  background: var(--danger-light);
  color: #991b1b;
  border: 1px solid #fca5a5;
  border-radius: 6px;
  padding: 0.5rem 1rem;
  font-size: 0.85rem;
}

.roll-area {
  border: 2px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  text-align: center;
  background: var(--bg-muted);
}

.roll-controls {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.roll-row {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  width: 100%;
  max-width: 400px;
}

.roll-hint {
  font-size: 0.8rem;
  color: var(--text-faint);
}

.pending-turn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.turn-header {
  font-size: 0.95rem;
  color: var(--text-secondary);
}

.turn-user {
  font-weight: 700;
  color: var(--accent);
}

.beer-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: var(--bg-card);
  border: 2px solid var(--accent);
  border-radius: var(--radius-lg);
  box-shadow: 0 4px 12px rgba(45, 106, 79, 0.1);
  max-width: 400px;
  width: 100%;
}

.beer-img {
  height: 80px;
  border-radius: 4px;
  flex-shrink: 0;
}

.beer-info {
  text-align: left;
}

.beer-title {
  font-weight: 700;
  font-size: 1rem;
  color: var(--text);
}

.beer-subtitle {
  font-size: 0.85rem;
  color: var(--text-muted);
}

.beer-producer {
  font-size: 0.8rem;
  color: var(--text-faint);
  margin-top: 0.2rem;
}

.veto-info {
  font-size: 0.85rem;
  color: var(--warning);
  font-weight: 500;
  background: var(--warning-light);
  padding: 0.4rem 0.8rem;
  border-radius: 6px;
}

.turn-actions {
  display: flex;
  gap: 0.75rem;
}

.admin-section {
  display: flex;
  justify-content: flex-end;
}

.vetoed-section {
  margin-top: 0.5rem;
}

.vetoed-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.vetoed-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  background: var(--danger-light);
  border-radius: 6px;
  border-left: 3px solid #f87171;
}

.consumed-section {
  margin-top: 0.5rem;
}

.section-label {
  font-size: 0.73rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-bottom: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.consumed-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.consumed-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0.75rem;
  background: var(--bg-muted);
  border-radius: 6px;
}

.consumed-thumb {
  height: 32px;
  border-radius: 2px;
  flex-shrink: 0;
}

.consumed-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}

.consumed-name {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-secondary);
}

.consumed-meta {
  font-size: 0.75rem;
  color: var(--text-faint);
}

.consumed-undo {
  font-size: 0.75rem;
  color: var(--border);
  cursor: pointer;
  margin-left: auto;
  flex-shrink: 0;
}

.consumed-undo:hover {
  color: var(--danger);
}

.empty-state {
  color: var(--text-muted);
  font-size: 0.875rem;
}
</style>
