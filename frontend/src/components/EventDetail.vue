<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import Select from 'primevue/select'
import RollGame from './RollGame.vue'
import { useAuthStore } from '../stores/auth'
import {
  getEvent, setEventLocked, inviteToEvent, uninviteFromEvent,
  importSharedListToEvent, removeBeerFromEvent, setScore, deleteScore,
  listAllUsers, listSharedLists, setEventHidden, toggleEventPublic,
  deleteEvent, archiveEvent, unarchiveEvent, updateEvent,
  type Event, type EventBeer, type ShareUser, type SharedList
} from '../api/client'

const props = defineProps<{ eventId: number }>()
const emit = defineEmits<{ back: [] }>()
const authStore = useAuthStore()

const event = ref<Event | null>(null)
const loading = ref(false)

// Invite dialog
const inviteDialogVisible = ref(false)
const allUsers = ref<ShareUser[]>([])
const inviteBusy = ref<Set<number>>(new Set())

// Import shared list
const sharedLists = ref<SharedList[]>([])
const selectedList = ref<SharedList | null>(null)
const importDialogVisible = ref(false)

// Score editing
const editingCell = ref<{ beerId: number; userId: number } | null>(null)
const editValue = ref<string>('')
const saving = ref(false)

// Event metadata editing
const editDialogVisible = ref(false)
const editDate = ref('')
const editDesc = ref('')
const editSaving = ref(false)

function openEditDialog() {
  if (!event.value) return
  editDate.value = event.value.eventDate || ''
  editDesc.value = event.value.description || ''
  editDialogVisible.value = true
}

async function saveEventMeta() {
  if (!event.value || editSaving.value) return
  editSaving.value = true
  try {
    await updateEvent(event.value.id, event.value.name, editDesc.value.trim(), editDate.value.trim())
    editDialogVisible.value = false
    await loadEvent()
  } catch (e: any) {
    alert(e?.message || 'Failed to update event')
  } finally {
    editSaving.value = false
  }
}

function isOwner() {
  return event.value?.ownerId === authStore.user?.id
}

function canEdit() {
  return isOwner() || authStore.user?.role === 'admin'
}

const isRoll = computed(() => event.value?.type === 'roll')
const isAdminUser = computed(() => authStore.user?.role === 'admin')
const isArchived = computed(() => !!event.value?.archivedAt)

const archivedDate = computed(() => {
  if (!event.value?.archivedAt) return ''
  try { return new Date(event.value.archivedAt).toLocaleString() } catch { return event.value.archivedAt }
})

async function toggleHidden() {
  if (!event.value) return
  await setEventHidden(event.value.id, !event.value.hidden)
  await loadEvent()
}

async function togglePublic() {
  if (!event.value) return
  await toggleEventPublic(event.value.id)
  await loadEvent()
}

const publicUrl = computed(() => {
  if (!event.value?.public) return null
  return `${window.location.origin}/roll`
})

function copyPublicUrl() {
  if (publicUrl.value) navigator.clipboard.writeText(publicUrl.value)
}

// All participants = owner + attendees
const participants = computed(() => {
  if (!event.value) return []
  const list: { userId: number; username: string }[] = []
  list.push({ userId: event.value.ownerId, username: event.value.ownerName })
  for (const a of event.value.attendees || []) {
    list.push({ userId: a.userId, username: a.username })
  }
  return list
})

function getScore(beerId: number, userId: number): number | null {
  const s = (event.value?.scores || []).find(s => s.eventBeerId === beerId && s.userId === userId)
  return s ? s.score : null
}

function beerAverage(beer: EventBeer): string {
  const scores = (event.value?.scores || []).filter(s => s.eventBeerId === beer.id)
  if (scores.length === 0) return '-'
  const avg = scores.reduce((sum, s) => sum + s.score, 0) / scores.length
  return avg.toFixed(1)
}

async function loadEvent() {
  loading.value = true
  try {
    event.value = await getEvent(props.eventId)
  } finally {
    loading.value = false
  }
}

async function toggleLock() {
  if (!event.value) return
  await setEventLocked(event.value.id, !event.value.locked)
  await loadEvent()
}

async function doDeleteEvent() {
  if (!event.value) return
  if (!confirm(`Delete event "${event.value.name}"? This cannot be undone.`)) return
  await deleteEvent(event.value.id)
  emit('back')
}

async function doArchive() {
  if (!event.value) return
  if (!confirm(`Archive event "${event.value.name}"? It will be hidden from all non-admin users. Data is preserved and only visible to admins.`)) return
  try {
    await archiveEvent(event.value.id)
    emit('back')
  } catch (e: any) {
    alert(e?.message || 'Failed to archive event')
  }
}

async function doUnarchive() {
  if (!event.value) return
  try {
    await unarchiveEvent(event.value.id)
    await loadEvent()
  } catch (e: any) {
    alert(e?.message || 'Failed to unarchive event')
  }
}

// Invite
async function openInviteDialog() {
  try { allUsers.value = await listAllUsers() } catch { allUsers.value = [] }
  inviteDialogVisible.value = true
}

function otherUsers(): ShareUser[] {
  return allUsers.value.filter(u => u.userId !== authStore.user?.id)
}

function isInvited(userId: number): boolean {
  return (event.value?.attendees || []).some(a => a.userId === userId)
}

async function toggleInvite(userId: number) {
  if (!event.value || inviteBusy.value.has(userId)) return
  inviteBusy.value.add(userId)
  try {
    if (isInvited(userId)) {
      await uninviteFromEvent(event.value.id, userId)
    } else {
      await inviteToEvent(event.value.id, userId)
    }
    await loadEvent()
  } finally {
    inviteBusy.value.delete(userId)
  }
}

// Import shared list
async function openImportDialog() {
  try { sharedLists.value = await listSharedLists() } catch { sharedLists.value = [] }
  selectedList.value = null
  importDialogVisible.value = true
}

async function doImport() {
  if (!event.value || !selectedList.value) return
  let replace = false
  if (isRoll.value && event.value.beerCount > 0) {
    if (!confirm('This will replace all beers currently in this roll event with the contents of the selected list. Continue?')) return
    replace = true
  }
  await importSharedListToEvent(event.value.id, selectedList.value.id, replace)
  importDialogVisible.value = false
  await loadEvent()
}

// Remove beer
async function doRemoveBeer(beerId: number) {
  if (!event.value) return
  await removeBeerFromEvent(event.value.id, beerId)
  await loadEvent()
}

// Scoring
function startEdit(beerId: number, userId: number) {
  if (event.value?.locked) return
  if (userId !== authStore.user?.id) return
  editingCell.value = { beerId, userId }
  const current = getScore(beerId, userId)
  editValue.value = current !== null ? String(current) : ''
  nextTick(() => {
    const input = document.querySelector('.score-input') as HTMLInputElement
    input?.focus()
    input?.select()
  })
}

async function saveScore() {
  if (saving.value || !editingCell.value || !event.value) return
  saving.value = true
  const { beerId } = editingCell.value
  editingCell.value = null
  try {
    const val = String(editValue.value).trim()
    if (val === '') {
      await deleteScore(event.value.id, beerId)
    } else {
      const num = parseInt(val, 10)
      if (isNaN(num) || num < 0 || num > 10) return
      await setScore(event.value.id, beerId, num)
    }
    await loadEvent()
  } catch (e) {
    console.error('Failed to save score:', e)
  } finally {
    saving.value = false
  }
}

function cancelEdit() {
  editingCell.value = null
}

function isEditingCell(beerId: number, userId: number) {
  return editingCell.value?.beerId === beerId && editingCell.value?.userId === userId
}

onMounted(loadEvent)
</script>

<template>
  <div class="card">
    <div v-if="loading && !event" class="empty-state" style="padding: 2rem;">Loading...</div>
    <template v-else-if="event">
      <!-- Header -->
      <div class="detail-header">
        <Button icon="pi pi-arrow-left" text size="small" @click="emit('back')" />
        <div class="detail-title">
          <h3>{{ event.name }}</h3>
          <span v-if="event.eventDate" class="detail-date">
            <i class="pi pi-calendar"></i> {{ event.eventDate }}
          </span>
        </div>
        <div class="detail-actions">
          <Button v-if="isRoll && isAdminUser" :label="event.public ? 'Unpublish' : 'Publish'" icon="pi pi-globe" size="small" :severity="event.public ? 'success' : 'secondary'" @click="togglePublic" />
          <Button v-if="isRoll && isAdminUser" :label="event.hidden ? 'Reveal' : 'Hide'" :icon="event.hidden ? 'pi pi-eye' : 'pi pi-eye-slash'" size="small" :severity="event.hidden ? 'success' : 'warn'" @click="toggleHidden" />
          <Button v-if="canEdit() && !isArchived" label="Edit" icon="pi pi-pencil" size="small" severity="secondary" @click="openEditDialog" />
          <Button v-if="canEdit()" :label="event.locked ? 'Unlock' : 'Lock'" :icon="event.locked ? 'pi pi-lock-open' : 'pi pi-lock'" size="small" :severity="event.locked ? 'warn' : 'secondary'" @click="toggleLock" />
          <Button v-if="isOwner()" label="Invite" icon="pi pi-user-plus" size="small" severity="secondary" @click="openInviteDialog" />
          <Button v-if="canEdit() && !event.locked && !isArchived" label="Import List" icon="pi pi-download" size="small" severity="secondary" @click="openImportDialog" />
          <Button v-if="canEdit() && !isArchived" label="Archive" icon="pi pi-box" size="small" severity="warn" @click="doArchive" />
          <Button v-if="isAdminUser && isArchived" label="Unarchive" icon="pi pi-replay" size="small" severity="success" @click="doUnarchive" />
          <Button v-if="isAdminUser && isArchived" label="Delete permanently" icon="pi pi-trash" size="small" severity="danger" @click="doDeleteEvent" />
          <Button v-if="canEdit() && !isArchived" label="Delete" icon="pi pi-trash" size="small" severity="danger" @click="doDeleteEvent" />
        </div>
      </div>

      <p v-if="event.description" class="detail-desc">{{ event.description }}</p>

      <div v-if="isArchived" class="archived-banner">
        <i class="pi pi-box"></i> Archived {{ archivedDate }} — visible to admins only.
      </div>

      <div v-if="event.hidden" class="hidden-banner">
        <i class="pi pi-eye-slash"></i> This event is hidden from participants.
      </div>

      <div v-if="event.locked" class="locked-banner">
        <i class="pi pi-lock"></i> Event is locked — scoring is closed.
      </div>

      <div v-if="publicUrl" class="public-banner">
        <i class="pi pi-globe"></i>
        Public link:
        <a :href="publicUrl" target="_blank" class="public-link">{{ publicUrl }}</a>
        <button class="copy-btn" @click="copyPublicUrl" title="Copy link">
          <i class="pi pi-copy"></i>
        </button>
      </div>

      <!-- Attendees summary -->
      <div class="attendee-chips">
        <span class="attendee-chip owner">{{ event.ownerName }} (host)</span>
        <span v-for="a in event.attendees" :key="a.userId" class="attendee-chip">{{ a.username }}</span>
      </div>

      <!-- Roll game (for roll events) -->
      <RollGame v-if="isRoll" :eventId="event.id" :participants="participants" :canEdit="canEdit()" />

      <!-- Scoring matrix (for tasting events) -->
      <div v-else-if="(event.beers || []).length > 0" class="matrix-wrapper">
        <table class="score-matrix">
          <thead>
            <tr>
              <th class="sticky-col">User</th>
              <th v-for="beer in event.beers" :key="beer.id" class="beer-header">
                <div class="beer-header-content">
                  <img v-if="beer.imageUrl" :src="beer.imageUrl.replace('_400.', '_60.')" class="beer-thumb" />
                  <span class="beer-name">{{ beer.productNameBold }}</span>
                  <span v-if="beer.productNameThin" class="beer-name-thin">{{ beer.productNameThin }}</span>
                  <i v-if="canEdit() && !event.locked" class="pi pi-times beer-remove" @click="doRemoveBeer(beer.id)" title="Remove beer"></i>
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in participants" :key="p.userId" :class="{ 'my-row': p.userId === authStore.user?.id }">
              <td class="sticky-col user-cell">{{ p.username }}</td>
              <td
                v-for="beer in event.beers" :key="beer.id"
                class="score-cell"
                :class="{ editable: p.userId === authStore.user?.id && !event.locked, editing: isEditingCell(beer.id, p.userId) }"
                @click="startEdit(beer.id, p.userId)"
              >
                <template v-if="isEditingCell(beer.id, p.userId)">
                  <input
                    v-model="editValue"
                    type="text" inputmode="numeric" pattern="[0-9]*"
                    class="score-input"
                    @blur="saveScore"
                    @keydown.enter.prevent="($event.target as HTMLInputElement).blur()"
                    @keydown.escape.prevent="cancelEdit"
                  />
                </template>
                <template v-else>
                  <span v-if="getScore(beer.id, p.userId) !== null" class="score-val">{{ getScore(beer.id, p.userId) }}</span>
                  <span v-else class="score-empty">-</span>
                </template>
              </td>
            </tr>
          </tbody>
          <tfoot>
            <tr>
              <td class="sticky-col avg-label">Average</td>
              <td v-for="beer in event.beers" :key="beer.id" class="avg-cell">
                {{ beerAverage(beer) }}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>

      <div v-else-if="!isRoll" class="empty-state">
        No beers yet. {{ canEdit() ? 'Import a list to get started.' : 'The host needs to add beers.' }}
      </div>
    </template>

    <!-- Invite dialog -->
    <Dialog v-model:visible="inviteDialogVisible" modal header="Invite Users" :style="{ width: '360px', maxWidth: '95vw' }">
      <div class="invite-body">
        <div v-if="otherUsers().length === 0" class="empty-state">No other users.</div>
        <label v-for="u in otherUsers()" :key="u.userId" class="invite-row" :class="{ busy: inviteBusy.has(u.userId) }">
          <Checkbox :modelValue="isInvited(u.userId)" :binary="true" :disabled="inviteBusy.has(u.userId)" @update:modelValue="toggleInvite(u.userId)" />
          <span class="invite-username">{{ u.username }}</span>
          <span v-if="isInvited(u.userId)" class="invite-status">Invited</span>
        </label>
      </div>
    </Dialog>

    <!-- Edit event dialog -->
    <Dialog v-model:visible="editDialogVisible" modal header="Edit Event" :style="{ width: '400px', maxWidth: '95vw' }">
      <div class="edit-body">
        <label class="edit-label">Date</label>
        <InputText v-model="editDate" placeholder="Date (e.g. 2026-04-10)" size="small" style="width: 100%;" />
        <label class="edit-label">Description</label>
        <InputText v-model="editDesc" placeholder="Description (optional)" size="small" style="width: 100%;" />
        <div class="edit-actions">
          <Button label="Cancel" size="small" severity="secondary" text @click="editDialogVisible = false" :disabled="editSaving" />
          <Button label="Save" icon="pi pi-check" size="small" @click="saveEventMeta" :disabled="editSaving" />
        </div>
      </div>
    </Dialog>

    <!-- Import list dialog -->
    <Dialog v-model:visible="importDialogVisible" modal header="Import List" :style="{ width: '400px', maxWidth: '95vw' }">
      <div class="import-body">
        <p style="margin: 0 0 0.75rem; font-size: 0.85rem; color: #6b7280;">
          Select a list to import its products:
        </p>
        <Select v-model="selectedList" :options="sharedLists" optionLabel="name" placeholder="Select list..." style="width: 100%;" />
        <Button label="Import" icon="pi pi-download" size="small" style="margin-top: 0.75rem;" @click="doImport" :disabled="!selectedList" />
      </div>
    </Dialog>
  </div>
</template>

<style scoped>
.detail-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.detail-title {
  flex: 1;
}

.detail-title h3 {
  margin: 0;
  font-size: 1.1rem;
}

.detail-date {
  font-size: 0.8rem;
  color: var(--text-muted);
}

.detail-date i {
  font-size: 0.7rem;
}

.detail-actions {
  display: flex;
  gap: 0.5rem;
}

.detail-desc {
  margin: 0 0 0.75rem;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.hidden-banner {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--purple-light);
  color: var(--purple);
  border: 1px solid #c4b5fd;
  border-radius: 6px;
  padding: 0.5rem 1rem;
  margin-bottom: 0.75rem;
  font-size: 0.85rem;
  font-weight: 500;
}

.archived-banner {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  background: #f3f4f6;
  color: #4b5563;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  padding: 0.5rem 1rem;
  margin-bottom: 0.75rem;
  font-size: 0.85rem;
  font-weight: 500;
}

.locked-banner {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  background: var(--warning-light);
  color: #92400e;
  border: 1px solid #f0c36d;
  border-radius: 6px;
  padding: 0.5rem 1rem;
  margin-bottom: 0.75rem;
  font-size: 0.85rem;
  font-weight: 500;
}

.public-banner {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  background: #e0f2fe;
  color: #0369a1;
  border: 1px solid #7dd3fc;
  border-radius: 6px;
  padding: 0.5rem 1rem;
  margin-bottom: 0.75rem;
  font-size: 0.85rem;
  font-weight: 500;
}

.public-link {
  color: #0369a1;
  text-decoration: underline;
  word-break: break-all;
}

.copy-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #0369a1;
  padding: 0.2rem;
  margin-left: 0.25rem;
}

.copy-btn:hover {
  color: #075985;
}

.attendee-chips {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}

.attendee-chip {
  background: var(--bg-muted);
  color: var(--text-secondary);
  border-radius: 4px;
  padding: 0.15rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
}

.attendee-chip.owner {
  background: var(--accent-light);
  color: var(--accent);
}

/* Scoring matrix */
.matrix-wrapper {
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius);
}

.score-matrix {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
  min-width: max-content;
}

.score-matrix thead th {
  background: var(--bg-muted);
  border-bottom: 2px solid var(--border);
  padding: 0.5rem;
  font-weight: 600;
  font-size: 0.75rem;
  text-align: center;
  vertical-align: bottom;
  min-width: 90px;
}

.sticky-col {
  position: sticky;
  left: 0;
  background: var(--bg-muted);
  z-index: 2;
  border-right: 2px solid var(--border);
  min-width: 110px;
  padding: 0.5rem 0.75rem;
  font-weight: 600;
}

.beer-header-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.2rem;
  position: relative;
}

.beer-thumb {
  height: 28px;
  border-radius: 2px;
}

.beer-name {
  font-size: 0.7rem;
  line-height: 1.2;
  text-align: center;
  word-break: break-word;
}

.beer-name-thin {
  font-size: 0.65rem;
  color: var(--text-faint);
  font-weight: 400;
}

.beer-remove {
  font-size: 0.6rem;
  color: var(--border);
  cursor: pointer;
  position: absolute;
  top: -2px;
  right: -2px;
}

.beer-remove:hover {
  color: var(--danger);
}

.score-matrix tbody td {
  border-bottom: 1px solid var(--border-light);
  padding: 0.4rem 0.5rem;
  text-align: center;
  vertical-align: middle;
}

.user-cell {
  text-align: left;
  font-weight: 600;
  font-size: 0.8rem;
}

.my-row td {
  background: var(--accent-light);
}

.my-row .sticky-col {
  background: var(--accent-faint);
}

.score-cell.editable {
  cursor: pointer;
  position: relative;
}

.score-cell.editable:hover {
  background: var(--accent-faint);
}

.score-cell.editable .score-empty:hover::after {
  content: 'click to score';
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: var(--text-secondary);
  color: #fff;
  font-size: 0.65rem;
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
  white-space: nowrap;
  pointer-events: none;
}

.score-val {
  font-weight: 600;
  font-size: 0.9rem;
}

.score-empty {
  color: var(--border);
}

.score-input {
  width: 40px;
  text-align: center;
  border: 2px solid var(--accent);
  border-radius: 4px;
  font-size: 0.9rem;
  font-weight: 600;
  font-family: var(--font);
  padding: 0.15rem;
  outline: none;
  color: var(--text);
}

.score-input::-webkit-inner-spin-button,
.score-input::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.score-matrix tfoot td {
  border-top: 2px solid var(--border);
  padding: 0.5rem;
  font-weight: 700;
  text-align: center;
  background: var(--bg-muted);
  font-size: 0.85rem;
}

.avg-label {
  text-align: left;
}

.empty-state {
  color: var(--text-muted);
  font-size: 0.875rem;
  padding: 0.75rem 0;
}

/* Invite dialog */
.invite-body {
  padding: 0.25rem 0;
}

.invite-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.5rem 0.6rem;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s;
}

.invite-row:hover {
  background: var(--bg-muted);
}

.invite-row.busy {
  opacity: 0.5;
  pointer-events: none;
}

.invite-username {
  font-size: 0.875rem;
  font-weight: 500;
  flex: 1;
}

.invite-status {
  font-size: 0.7rem;
  color: var(--purple);
  font-weight: 500;
}

.import-body {
  padding: 0.5rem 0;
}

.edit-body {
  padding: 0.25rem 0;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.edit-label {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-top: 0.5rem;
}

.edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.75rem;
}
</style>
