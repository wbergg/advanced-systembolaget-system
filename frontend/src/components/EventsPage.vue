<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import EventDetail from './EventDetail.vue'
import { useAuthStore } from '../stores/auth'
import { listEvents, createEvent, deleteEvent, type Event } from '../api/client'

const authStore = useAuthStore()
const events = ref<Event[]>([])
const activeEvent = ref<Event | null>(null)
const newName = ref('')
const newDate = ref('')
const newDesc = ref('')
const newType = ref<'tasting' | 'roll'>('tasting')

const isAdmin = computed(() => authStore.user?.role === 'admin')

function isOwner(ev: Event) {
  return ev.ownerId === authStore.user?.id
}

function myEvents() {
  return events.value.filter(e => isOwner(e))
}

function invitedEvents() {
  return events.value.filter(e => !isOwner(e))
}

async function loadEvents() {
  events.value = await listEvents()
}

async function doCreate() {
  if (!newName.value.trim()) return
  const ev = await createEvent(
    newName.value.trim(),
    newDesc.value.trim(),
    newDate.value.trim(),
    newType.value,
  )
  newName.value = ''
  newDate.value = ''
  newDesc.value = ''
  newType.value = 'tasting'
  await loadEvents()
  activeEvent.value = ev
}

async function doDelete(id: number) {
  await deleteEvent(id)
  await loadEvents()
}

function openEvent(ev: Event) {
  activeEvent.value = ev
}

function goBack() {
  activeEvent.value = null
  loadEvents()
}

onMounted(loadEvents)
</script>

<template>
  <div v-if="activeEvent">
    <EventDetail :eventId="activeEvent.id" @back="goBack" />
  </div>
  <div v-else class="card">
    <div class="card-header">
      <h3>Events</h3>
    </div>

    <!-- Create form -->
    <div class="event-create">
      <div class="event-create-row">
        <InputText v-model="newName" placeholder="Event name..." size="small" @keyup.enter="doCreate" style="flex: 2;" />
        <InputText v-model="newDate" placeholder="Date (e.g. 2026-04-10)" size="small" style="flex: 1;" />
      </div>
      <div class="event-create-row">
        <label class="type-toggle">
          <input type="radio" value="tasting" v-model="newType" /> Tasting
        </label>
        <label class="type-toggle">
          <input type="radio" value="roll" v-model="newType" /> Roll
        </label>
      </div>
      <div class="event-create-row">
        <InputText v-model="newDesc" placeholder="Description (optional)" size="small" style="flex: 1;" />
        <Button label="Create Event" icon="pi pi-plus" size="small" @click="doCreate" :disabled="!newName.trim()" />
      </div>
    </div>

    <!-- My Events -->
    <div class="section-label">
      <i class="pi pi-calendar" style="font-size: 0.75rem"></i> My Events
    </div>
    <div v-if="myEvents().length === 0" class="empty-state">No events yet. Create one above.</div>
    <div class="event-list">
      <div v-for="ev in myEvents()" :key="ev.id" class="event-card" @click="openEvent(ev)">
        <div class="event-card-header">
          <i v-if="ev.type === 'roll'" class="pi pi-sync" style="color: var(--purple); font-size: 0.75rem;" title="Roll event"></i>
          <span class="event-card-name">{{ ev.name }}</span>
          <span v-if="ev.hidden" class="hidden-badge">Hidden</span>
          <i v-if="ev.locked" class="pi pi-lock" style="color: var(--warning); font-size: 0.75rem;" title="Locked"></i>
          <i class="pi pi-trash action-icon red" @click.stop="doDelete(ev.id)" title="Delete"></i>
        </div>
        <div class="event-card-meta">
          <span v-if="ev.eventDate"><i class="pi pi-calendar"></i> {{ ev.eventDate }}</span>
          <span><i class="pi pi-users"></i> {{ ev.attendeeCount }} invited</span>
          <span><i class="pi pi-list"></i> {{ ev.beerCount }} beers</span>
        </div>
        <div v-if="ev.description" class="event-card-desc">{{ ev.description }}</div>
      </div>
    </div>

    <!-- Invited Events -->
    <div v-if="invitedEvents().length > 0">
      <div class="section-label" style="color: var(--purple); margin-top: 1rem;">
        <i class="pi pi-users" style="font-size: 0.75rem"></i> Invited Events
      </div>
      <div class="event-list">
        <div v-for="ev in invitedEvents()" :key="ev.id" class="event-card invited" @click="openEvent(ev)">
          <div class="event-card-header">
            <i v-if="ev.type === 'roll'" class="pi pi-sync" style="color: var(--purple); font-size: 0.75rem;" title="Roll event"></i>
            <span class="event-card-name">{{ ev.name }}</span>
            <span class="owner-badge">{{ ev.ownerName }}</span>
            <i v-if="ev.locked" class="pi pi-lock" style="color: var(--warning); font-size: 0.75rem;" title="Locked"></i>
            <i v-if="isAdmin" class="pi pi-trash action-icon red" @click.stop="doDelete(ev.id)" title="Delete"></i>
          </div>
          <div class="event-card-meta">
            <span v-if="ev.eventDate"><i class="pi pi-calendar"></i> {{ ev.eventDate }}</span>
            <span><i class="pi pi-users"></i> {{ ev.attendeeCount }} invited</span>
            <span><i class="pi pi-list"></i> {{ ev.beerCount }} beers</span>
          </div>
          <div v-if="ev.description" class="event-card-desc">{{ ev.description }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.event-create {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 1.25rem;
  padding: 1rem;
  border: 1px solid var(--border-light);
  border-radius: var(--radius);
  background: var(--bg-muted);
}

.event-create-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
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

.empty-state {
  color: var(--text-muted);
  font-size: 0.875rem;
  padding: 0.5rem 0;
}

.event-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.event-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 0.75rem 1rem;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--bg-card);
}

.event-card:hover {
  border-color: var(--accent);
  background: var(--accent-light);
}

.event-card.invited {
  border-left: 3px solid var(--purple);
  background: var(--purple-light);
}

.event-card.invited:hover {
  background: var(--purple-light);
}

.event-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.event-card-name {
  font-weight: 600;
  font-size: 0.95rem;
}

.owner-badge {
  background: var(--purple-light);
  color: var(--purple);
  border-radius: 4px;
  padding: 0.1rem 0.4rem;
  font-size: 0.7rem;
  font-weight: 500;
}

.event-card-meta {
  display: flex;
  gap: 1rem;
  margin-top: 0.35rem;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.event-card-meta i {
  font-size: 0.7rem;
  margin-right: 0.2rem;
}

.event-card-desc {
  margin-top: 0.3rem;
  font-size: 0.8rem;
  color: var(--text-faint);
}

.action-icon {
  font-size: 0.7rem;
  color: var(--text-faint);
  cursor: pointer;
  margin-left: auto;
}

.action-icon.red:hover {
  color: var(--danger);
}

.type-toggle {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.85rem;
  cursor: pointer;
  color: var(--text-secondary);
}

.type-toggle input {
  accent-color: var(--accent);
}

.hidden-badge {
  background: var(--purple-light);
  color: var(--purple);
  border-radius: 4px;
  padding: 0.1rem 0.4rem;
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
}
</style>
