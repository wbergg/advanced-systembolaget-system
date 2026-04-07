<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from 'primevue/button'
import { updateNote, getComments, addComment, deleteComment, type Product, type Comment } from '../api/client'
import { useAuthStore } from '../stores/auth'

const props = defineProps<{ product: Product }>()
const emit = defineEmits<{ updated: [] }>()

const authStore = useAuthStore()
const note = ref(props.product.note || '')
const saving = ref(false)

const comments = ref<Comment[]>([])
const newComment = ref('')
const posting = ref(false)
const loadingComments = ref(false)

async function saveNote() {
  saving.value = true
  try {
    await updateNote(props.product.productId, note.value)
    emit('updated')
  } catch (e) {
    console.error('Failed to save note:', e)
  } finally {
    saving.value = false
  }
}

async function loadComments() {
  loadingComments.value = true
  try {
    comments.value = await getComments(props.product.productId)
  } catch (e) {
    console.error('Failed to load comments:', e)
  } finally {
    loadingComments.value = false
  }
}

async function postComment() {
  if (!newComment.value.trim()) return
  posting.value = true
  try {
    const c = await addComment(props.product.productId, newComment.value.trim())
    comments.value.unshift(c)
    newComment.value = ''
  } catch (e) {
    console.error('Failed to post comment:', e)
  } finally {
    posting.value = false
  }
}

async function removeComment(id: number) {
  try {
    await deleteComment(id)
    comments.value = comments.value.filter(c => c.id !== id)
  } catch (e) {
    console.error('Failed to delete comment:', e)
  }
}

onMounted(loadComments)
</script>

<template>
  <div class="product-detail">
    <div>
      <img v-if="product.imageUrl" :src="product.imageUrl" :alt="product.productNameBold" />
    </div>
    <div class="product-info">
      <h3>{{ product.productNameBold }} <span v-if="product.productNameThin" class="name-thin">{{ product.productNameThin }}</span></h3>
      <p><strong>Producer:</strong> {{ product.producerName }}</p>
      <p><strong>Price:</strong> {{ product.price }} kr &middot; <strong>Volume:</strong> {{ product.volumeText }} &middot; <strong>ABV:</strong> {{ product.alcoholPercentage }}%</p>
      <p><strong>Category:</strong> {{ product.categoryLevel1 }} / {{ product.categoryLevel2 }}</p>
      <p><strong>Country:</strong> {{ product.country }}</p>
      <p v-if="product.assortmentText"><strong>Assortment:</strong> {{ product.assortmentText }}</p>
      <p v-if="product.taste"><strong>Taste:</strong> {{ product.taste }}</p>
      <p v-if="product.usage"><strong>Usage:</strong> {{ product.usage }}</p>
      <p v-if="product.vintage"><strong>Vintage:</strong> {{ product.vintage }}</p>
      <p>
        <strong>Nr:</strong> {{ product.productNumber }}
        <span v-if="product.isOrganic" class="badge-organic" style="margin-left: 0.5rem">Organic</span>
        <span v-if="product.isNews" class="badge-news" style="margin-left: 0.5rem">New</span>
      </p>

      <div class="note-editor" style="max-width: 50%;">
        <label class="section-label">Notes</label>
        <textarea v-model="note" placeholder="Add a personal note about this product..."></textarea>
        <Button label="Save note" icon="pi pi-save" :loading="saving" @click="saveNote" size="small" style="margin-top: 0.4rem" />
      </div>

      <div class="comments-section" style="max-width: 50%;">
        <label class="section-label">Comments</label>
        <div class="comment-input">
          <textarea v-model="newComment" placeholder="Write a comment..." rows="2"></textarea>
          <Button label="Post" icon="pi pi-send" :loading="posting" :disabled="!newComment.trim()" @click="postComment" size="small" style="margin-top: 0.4rem" />
        </div>
        <div v-if="loadingComments" class="comments-empty">Loading comments...</div>
        <div v-else-if="comments.length === 0" class="comments-empty">No comments yet.</div>
        <div v-else class="comment-list">
          <div v-for="c in comments" :key="c.id" class="comment-item">
            <div class="comment-header">
              <strong>{{ c.username }}</strong>
              <span class="comment-time">{{ new Date(c.createdAt).toLocaleString() }}</span>
              <Button v-if="authStore.isAdmin" icon="pi pi-trash" severity="danger" text size="small" @click="removeComment(c.id)" style="margin-left: auto; padding: 0.2rem" />
            </div>
            <div class="comment-body">{{ c.comment }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.name-thin {
  color: var(--text-muted);
  font-weight: normal;
}

.section-label {
  font-weight: 600;
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.comments-section {
  margin-top: 0.75rem;
  border-top: 1px solid var(--border);
  padding-top: 0.5rem;
}

.comment-input textarea {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-family: var(--font);
  font-size: 0.85rem;
  resize: vertical;
  margin-top: 0.25rem;
  transition: border-color 0.2s;
  background: var(--bg-card);
  color: var(--text);
}

.comment-input textarea:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 2px rgba(45, 106, 79, 0.12);
}

.comments-empty {
  color: var(--text-muted);
  font-size: 0.8rem;
  margin-top: 0.5rem;
}

.comment-list {
  margin-top: 0.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.comment-item {
  padding: 0.4rem 0.6rem;
  background: var(--bg-muted);
  border: 1px solid var(--border-light);
  border-radius: 6px;
  font-size: 0.85rem;
}

.comment-header {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  margin-bottom: 0.15rem;
}

.comment-time {
  color: var(--text-faint);
  font-size: 0.75rem;
}

.comment-body {
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--text-secondary);
}
</style>
