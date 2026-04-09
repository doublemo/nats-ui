import { createRouter, createWebHistory } from 'vue-router'
import Dashboard from '../views/Dashboard.vue'
import JetStreamView from '../views/JetStreamView.vue'
import KVManager from '../views/KVManager.vue'
import ConnectionManager from '../views/ConnectionManager.vue'

const routes = [
  { path: '/', redirect: '/dashboard' },
  { path: '/connections', component: ConnectionManager },
  { path: '/dashboard', component: Dashboard },
  { path: '/jetstream', component: JetStreamView },
  { path: '/kv', component: KVManager },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})
