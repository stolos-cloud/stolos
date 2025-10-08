<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('dashboard.title')"
            :subheading="$t('dashboard.subheading')"
            :actions="actions"
        />
        <!-- Active connected nodes -->
        <v-sheet border rounded class="mt-4">
          <v-data-table
            :headers="nodeHeaders"
            :items="nodes"
            :items-length="nodes.length"
            :search="search"
            :loading="loading"
            :loading-text="$t('dashboard.onPremises.table.loadingText')"
            :no-data-text="$t('dashboard.onPremises.table.noDataText')"
            :items-per-page="10"
            :items-per-page-text="$t('dashboard.onPremises.table.itemsPerPageText')"
            class="elevation-8"
            mobile-breakpoint="md"
            :hide-default-footer="nodes.length < 10"
          >
            <!-- Slot for top -->
            <template v-slot:top>
              <v-toolbar>
                <v-toolbar-title>
                  {{ $t('dashboard.onPremises.table.title') }}
                </v-toolbar-title>
              </v-toolbar>
              <v-text-field
                v-model="search"
                label="Search"
                prepend-inner-icon="mdi-magnify"
                variant="outlined"
                hide-details
                single-line
                dense
                class="pa-3"
              />
            </template>

            <!-- Slot for status -->
            <template #item.status="{ item }">
                <v-chip color="success">
                    {{ item.status }}
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
import { downloadISO, createSamplesNodes, getConnectedNodes } from '@/services/provisioning.service';
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
const actions = computed(() => [
  {
    text: t('dashboard.buttons.downloadISOOnPremise'),
    color: 'primary',
    onClick: () => dialogDownloadISOOnPremise.value = true
  }
]);
const nodeHeaders = computed(() => [
  { title: t('dashboard.onPremises.table.headers.nodename'), value: 'name' },
  { title: t('dashboard.onPremises.table.headers.role'), value: 'role' },
  { title: t('dashboard.onPremises.table.headers.status'), value: 'status', align: "center" }
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
function cancelDownloadISO() {
    isoRadioButtons.value = undefined;
    dialogDownloadISOOnPremise.value = false;
}

function confirmDownloadISO() {
    if (!isValid.value) return;

    createSamplesNodes();
    downloadISO({ iso: isoRadioButtons.value })
    .then(({ data, headers }) => {
      let filename = "fallback.iso";
      const blob = new Blob([data], { type: headers['content-type'] });
      const url = URL.createObjectURL(blob);
      const contentDisposition = headers["content-disposition"];

      if (contentDisposition) {
        const match = contentDisposition.match(/filename="?(.+)"?/);
        if (match && match[1]) filename = match[1];
      }

      const link = document.createElement('a');
      link.href = url;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    })
    .catch(error => {
        console.error("Error downloading ISO:", error);
    })
    .finally(() => {
        isoRadioButtons.value = undefined;
        dialogDownloadISOOnPremise.value = false;
    });
}

function fetchConnectedNodesActive() {
    loading.value = true;

    getConnectedNodes({status: "active"})
    .then(response => {        
        nodes.value = response
            .filter(node => node.provider?.toLowerCase() === "onprem")
            .map(node => ({
                ...node,
                status: node.status.charAt(0).toUpperCase() + node.status.slice(1),
                role: node.role.charAt(0).toUpperCase() + node.role.slice(1)
            }));
    })
    .catch(error => {
        console.error('Error fetching connected nodes active:', error);
    })
    .finally(() => {
        loading.value = false;
    });
}
</script>