<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('dashboard.title')"
            :subheading="$t('dashboard.subheading')"
        />
        <v-sheet class="mt-4 border rounded">
          <v-data-table
            :headers="nodeHeaders"
            :items="nodes"
            :items-length="nodes.length"
            :search="search"
            :loading="loading"
            :loading-text="$t('dashboard.provision.table.loadingText')"
            :no-data-text="$t('dashboard.provision.table.noDataText')"
            :items-per-page="10"
            :items-per-page-text="$t('dashboard.provision.table.itemsPerPageText')"
            :hide-default-footer="nodes.length < 10"
            mobile-breakpoint="md"
          >
            <!-- Slot for top -->
            <template v-slot:top>
              <v-toolbar flat>
                <v-toolbar-title>
                  {{ $t('dashboard.provision.table.title') }}
                </v-toolbar-title>
                <BaseButton 
                  icon="mdi-download"
                  elevation="2"
                  :tooltip="$t('dashboard.buttons.downloadISOOnPremise')"
                  :text="$t('dashboard.buttons.downloadISOOnPremise')" 
                  @click="showDownloadISODialog" 
                />
              </v-toolbar>
              <v-text-field v-model="search" label="Search" prepend-inner-icon="mdi-magnify" variant="outlined" hide-details single-line dense class="pa-3"/>
            </template>

            <!-- Slot for status -->
            <template #item.status="{ item }">
                <v-chip :color="getStatusColor(item.status)">
                    {{ item.status }}
                </v-chip>
            </template>

            <!-- Slot for labels -->
            <template #item.labels="{ item }">
                <v-chip 
                  v-for="(label, index) in item.labels"
                  :key="index"
                  class="ma-1"
                >
                  {{ label }}
                </v-chip>
            </template>
          </v-data-table>
        </v-sheet>
        <BaseDialog v-model="dialogDownloadISOOnPremise" :title="$t('dashboard.dialogs.downloadISOOnPremise.title')" width="600" closable>
          <v-form v-model="isValid">
            <BaseNotice type="info" :text="$t('dashboard.dialogs.downloadISOOnPremise.noticeText')" />
            <BaseRadioButtons :RadioGroup="isoRadioButtons" />
          </v-form>
          <template #actions>
            <BaseButton size="small" variant="outlined" :text="$t('dashboard.buttons.cancel')" @click="cancelDownloadISO" />
            <BaseButton size="small" :text="$t('dashboard.buttons.download')" @click="confirmDownloadISO" :disabled="!isValid"/>
          </template>
        </BaseDialog>
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';
import BaseButton from '@/components/base/BaseButton.vue';
import BaseDialog from '@/components/base/BaseDialog.vue';
import BaseNotice from '@/components/base/BaseNotice.vue';
import BaseRadioButtons from '@/components/base/BaseRadioButtons.vue';
import { RadioGroup } from '@/models/RadioGroup.js';
import { generateISO, getConnectedNodes } from '@/services/provisioning.service';
import { computed, ref, reactive, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useStore } from 'vuex';

const { t } = useI18n();
const store = useStore();

// State
const dialogDownloadISOOnPremise = ref(false);
const isValid = ref(false);
const search = ref('');
const loading = ref(false);
const nodes = ref([]);

// Computed
const nodeHeaders = computed(() => [
  { title: t('dashboard.provision.table.headers.nodename'), value: 'name' },
  { title: t('dashboard.provision.table.headers.role'), value: 'role', align : "center" },
  { title: t('dashboard.provision.table.headers.provider'), value: 'provider', align : "center" },
  { title: t('dashboard.provision.table.headers.status'), value: 'status', align: "center" },
  { title: t('dashboard.provision.table.headers.labels'), value: 'labels', align: "center" },
]);
const listISOTypes = computed(() => store.getters['referenceLists/getIsoTypes']);

// Reactives
const isoRadioButtons = reactive(new RadioGroup({
  label: "ISO choice",
  precision: "Choose the architecture of the ISO you want to download",
  options: listISOTypes.value,
  required: true,
  rules: [(v) => !!v || t('dashboard.dialogs.downloadISOOnPremise.radioOptions.required')]
}));

//mounted
onMounted(() => {
    fetchConnectedNodesActive();
});

// Methods
function showDownloadISODialog() {
    dialogDownloadISOOnPremise.value = true;
}
function cancelDownloadISO() {
    isoRadioButtons.value = undefined;
    dialogDownloadISOOnPremise.value = false;
}

function confirmDownloadISO() {
    if (!isValid.value) return;

    generateISO({
        architecture: isoRadioButtons.value
    })
    .then(({ download_url, filename }) => {
        const link = document.createElement('a');
        link.href = download_url;
        link.download = filename;
        link.target = '_blank';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
    })
    .catch(error => {
        console.error("Error generating ISO:", error);
    })
    .finally(() => {
        isoRadioButtons.value = undefined;
        dialogDownloadISOOnPremise.value = false;
    });
}

function fetchConnectedNodesActive() {
    loading.value = true;

    getConnectedNodes()
    .then(response => {
        nodes.value = response
            .filter(node => node.status?.toLowerCase() !== "pending")
            .map(node => ({
                ...node,
                status: node.status.charAt(0).toUpperCase() + node.status.slice(1),
                role: node.role.charAt(0).toUpperCase() + node.role.slice(1),
                provider: node.provider.charAt(0).toUpperCase() + node.provider.slice(1),
                labels: JSON.parse(node.labels || '[]'),
            }));
    })
    .catch(error => {
        console.error('Error fetching connected nodes active:', error);
    })
    .finally(() => {
        loading.value = false;
    });
}
function getStatusColor(status) {
  switch (status.toLowerCase()) {
    case 'active':
      return 'success';
    case 'provisioning':
      return 'warning';
    case 'failed':
      return 'error';
    default:
      return 'grey';
  }
}
</script>
