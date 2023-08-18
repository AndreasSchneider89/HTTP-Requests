<template>
  <q-layout view="hHh lpR lFf">
    <div style="position:absolute;width:100vw;height:100vh;background-color:rgba(0,0,0,.2);z-index:9999;left:0;top:0;" v-if="isLoading">LOADING</div>

    <q-header elevated class="bg-primary text-white">
      <img alt="Vue logo" class="logo" src="@/assets/logo.svg" width="32" height="32" />
    </q-header>

    <q-drawer v-model="leftDrawerOpen" side="left" bordered overlay behavior="desktop">
      <ul>
        <li v-for="req in requests" @click="setActiveRequest(req)" :class="isActive(req.id)">{{ req.id }}</li>
      </ul>
    </q-drawer>

    <q-page-container>
      <h5 class="q-mt-lg">Request: {{ activeRequest.url }}</h5>
      <p>{{ activeRequest }}</p>
      <q-table
        class="q-mt-md"
        title="HTTP-Request"
        :rows="formattedRows"
        :columns="columns"
        row-key="name"
      />
    </q-page-container>
  </q-layout>
</template>

<script setup>
import { ref, computed } from 'vue';
import RequestDetail from './components/RequestDetail.vue';

const isLoading = ref(true);
const requests = ref([]);
const activeRequestId = ref('');
const activeRequest = ref({});
const leftDrawerOpen = ref(true);

function addRequest(receivedReq) {
  requests.value.unshift(receivedReq);
}

// Funktion zum Abonnieren von SSE-Ereignissen
function subscribeToSSE() {
  const evtSource = new EventSource("http://localhost:8081/sse");

  evtSource.onmessage = (e) => {
    const newRequest = JSON.parse(e.data);

    // Füge den neuen Request der Liste hinzu
    addRequest(newRequest)
    // Wähle den neuen Request aus, wenn bisher keiner ausgewählt ist
    if (!activeRequest.value.id) {
      setActiveRequest(newRequest);
    }
  };

  evtSource.onerror = (e) => {
    debugger;
  }

  evtSource.onopen = () => {
  }
}

// Aufruf der SSE-Abonnement-Funktion
subscribeToSSE();

fetch('http://localhost:8081/view-requests?p=1')
  .then((data) => {
    isLoading.value = false;
    data.json()
      .then((d) => {
        requests.value = d;
        activeRequest.value = d.length > 0 ? d[0] : {};
      })
      .catch((e) => {
        console.warn(e);
      });
  })
  .catch((err) => {
    isLoading.value = false;
  });

function toggleLeftDrawer() {
  leftDrawerOpen.value = !leftDrawerOpen.value;
}

function setActiveRequest(request) {
  activeRequest.value = request;
}

function isActive(id) {
  return activeRequest.value.id === id ? 'active' : 'inactive';
}

const columns = [
  {
    name: 'name',
    required: true,
    label: 'Type',
    align: 'left',
    field: (row) => row.name,
    format: (val) => `${val}`,
    sortable: true,
  },
  { name: 'calories', align: 'center', label: 'Value', field: 'calories', sortable: true },
];

const formattedRows = computed(() => {
  return [
    { name: 'id', calories: activeRequest.value.id },
    { name: 'method', calories: activeRequest.value.method },
    { name: 'url', calories: activeRequest.value.url },
    { name: 'timestamp', calories: activeRequest.value.timestamp },
    { name: 'remote_addr', calories: activeRequest.value.remote_addr },
    { name: 'user_agent', calories: activeRequest.value.user_agent },
    { name: 'content_type', calories: activeRequest.value.content_type },
    { name: 'body_params', calories: JSON.stringify(activeRequest.value.body_params) },
    { name: 'link_to_file', calories: activeRequest.value.link_to_file },
  ];
});
</script>

<style scoped>
.active {
  color: red;
}
</style>
