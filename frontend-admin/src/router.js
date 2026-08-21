import { createRouter, createWebHistory } from 'vue-router'
import TaskConfig from './views/TaskConfig.vue'
import LiveMonitor from './views/LiveMonitor.vue'
import ReportList from './views/ReportList.vue'
import ReportDetail from './views/ReportDetail.vue'
import Nodes from './views/Nodes.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'task', component: TaskConfig },
    { path: '/live', name: 'live', component: LiveMonitor },
    { path: '/reports', name: 'reports', component: ReportList },
    { path: '/reports/:id', name: 'report', component: ReportDetail },
    { path: '/nodes', name: 'nodes', component: Nodes },
  ],
})

export default router
