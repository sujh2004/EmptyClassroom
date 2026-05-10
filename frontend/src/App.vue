<template>
  <van-config-provider>
    <main class="app-shell">
      <header class="app-header">
        <div class="header-top">
          <h1 class="app-title">空教室</h1>
          <span class="header-date">{{ summary.date || today }}</span>
        </div>
        <div class="campus-switcher">
          <button
            v-for="campus in campuses"
            :key="campus.id"
            :class="['campus-btn', { active: activeTab === campus.id }]"
            @click="activeTab = campus.id"
          >
            {{ campus.name }}
          </button>
          <button class="campus-btn refresh" @click="load(true)" aria-label="刷新">↻</button>
        </div>
      </header>

      <section class="slot-picker">
        <div class="slot-picker-header">
          <span class="slot-picker-title">选择时段</span>
          <span class="slot-picker-hint">{{ selectedSlotsLabel }}</span>
        </div>
        <div class="slot-chips">
          <button
            v-for="slot in slots"
            :key="slot.number"
            :class="['chip', { selected: selectedSlots.includes(slot.number), current: slot.number === summary.current_slot }]"
            @click="toggleSlot(slot.number)"
          >
            {{ slot.number }}
          </button>
        </div>
      </section>

      <van-pull-refresh v-model="refreshing" @refresh="load(true)">
        <section class="content">
          <van-loading v-if="loading" color="#4f46e5" class="loading" />
          <van-empty v-else-if="error" image="error" :description="error" />
          <van-empty v-else-if="!summary.buildings.length" description="暂无数据" />

          <div v-else class="building-list">
            <article
              v-for="building in summary.buildings"
              :key="building.building"
              class="building-card"
              @click="openBuilding(building)"
            >
              <div class="building-main">
                <h2>{{ building.building }}</h2>
                <span class="building-ratio">{{ building.free_rooms }}/{{ building.total_rooms }}</span>
              </div>
              <div class="meter">
                <span :style="{ width: `${building.free_ratio}%` }" />
              </div>
            </article>
          </div>
        </section>
      </van-pull-refresh>

      <van-popup v-model:show="showRooms" position="bottom" :style="{ height: '80%' }">
        <section class="room-sheet" v-if="selectedBuilding">
          <header class="room-sheet-header">
            <div>
              <h2>{{ selectedBuilding.building }}</h2>
              <p>{{ selectedBuilding.free_rooms }} 间空闲</p>
            </div>
            <button class="close-btn" @click="showRooms = false">×</button>
          </header>

          <div class="room-list">
            <article v-for="room in selectedBuilding.rooms" :key="room.id || room.room_number" class="room-card">
              <div class="room-title">
                <strong>{{ room.room_number }}</strong>
                <span :class="['room-state', room.free_now ? 'free' : 'busy']">
                  {{ room.free_now ? '空闲' : '占用' }}
                </span>
              </div>
              <div class="slot-grid">
                <span
                  v-for="slot in slots"
                  :key="slot.number"
                  :class="slotClass(room.occupancy, slot.number)"
                >
                  {{ slot.number }}
                </span>
              </div>
            </article>
          </div>
        </section>
      </van-popup>
    </main>
  </van-config-provider>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { showToast } from 'vant'
import { fetchClassrooms } from './api'

const campuses = [
  { id: 0, name: '西土城' },
  { id: 1, name: '沙河' }
]

const slots = [
  { number: 1, start: '08:00', end: '08:45' },
  { number: 2, start: '08:50', end: '09:35' },
  { number: 3, start: '09:50', end: '10:35' },
  { number: 4, start: '10:40', end: '11:25' },
  { number: 5, start: '11:30', end: '12:15' },
  { number: 6, start: '13:00', end: '13:45' },
  { number: 7, start: '13:50', end: '14:35' },
  { number: 8, start: '14:45', end: '15:30' },
  { number: 9, start: '15:40', end: '16:25' },
  { number: 10, start: '16:35', end: '17:20' },
  { number: 11, start: '17:25', end: '18:10' },
  { number: 12, start: '18:30', end: '19:15' },
  { number: 13, start: '19:20', end: '20:05' },
  { number: 14, start: '20:10', end: '20:55' }
]

const activeTab = ref(0)
const selectedSlots = ref([resolveInitialSlot()])
const summary = ref({ date: '', current_slot: 0, selected_slots: [], buildings: [] })
const loading = ref(false)
const refreshing = ref(false)
const error = ref('')
const showRooms = ref(false)
const selectedBuilding = ref(null)

const today = new Date().toISOString().slice(0, 10)
const activeCampus = computed(() => campuses[activeTab.value] || campuses[0])

const selectedSlotsLabel = computed(() => {
  if (selectedSlots.value.length === 0) return '全天空闲'
  const sorted = [...selectedSlots.value].sort((a, b) => a - b)
  return sorted.map(s => `第${s}节`).join('、')
})

function toggleSlot(number) {
  const idx = selectedSlots.value.indexOf(number)
  if (idx >= 0) {
    selectedSlots.value.splice(idx, 1)
  } else {
    selectedSlots.value.push(number)
  }
}

watch(activeTab, () => {
  showRooms.value = false
  load(false)
})

watch(selectedSlots, () => {
  showRooms.value = false
  load(false)
}, { deep: true })

onMounted(() => load(false))

async function load(forceToast) {
  loading.value = !refreshing.value
  error.value = ''
  try {
    summary.value = await fetchClassrooms(activeCampus.value.id, selectedSlots.value)
    if (forceToast) showToast('已刷新')
  } catch (err) {
    error.value = err.message || '加载失败'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

function openBuilding(building) {
  selectedBuilding.value = building
  showRooms.value = true
}

function slotClass(occupancy, slot) {
  const busy = occupancy?.[slot - 1] === '1'
  return {
    slot: true,
    'is-free': !busy,
    'is-busy': busy,
    'is-current': slot === summary.value.current_slot,
    'is-selected': selectedSlots.value.includes(slot)
  }
}

function resolveInitialSlot() {
  const now = new Date()
  const minutes = now.getHours() * 60 + now.getMinutes()
  for (const slot of slots) {
    const [sh, sm] = slot.start.split(':').map(Number)
    const [eh, em] = slot.end.split(':').map(Number)
    if (minutes >= sh * 60 + sm && minutes < eh * 60 + em) return slot.number
    if (minutes < sh * 60 + sm) return slot.number
  }
  return 1
}
</script>
