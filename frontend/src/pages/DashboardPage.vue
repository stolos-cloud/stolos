<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('dashboard.title')"
            :subheading="$t('dashboard.subheading')"
            :actions="actions"
        />
        <!-- Tableau d'état final des noeuds -->
        <div class="mt-4">
            <h3>{{ $t('provisioning.nodeStatesTitle') }}</h3>
            <v-data-table
                :headers="stateHeaders"
                :items="nodeStates"
                class="elevation-1 mt-2"
            ></v-data-table>
            <BaseButton color="primary" class="mt-2" @click="goDashboard">
                {{ $t('provisioning.dashboardButton') }}
            </BaseButton>
        </div>
        <BaseDialog v-model="dialogDownloadISOOnPremise" :title="$t('dashboard.dialogs.downloadISOOnPremise.title')" width="600" closable>
          <v-form v-model="isValid">
            <BaseNotice type="info" :text="$t('dashboard.dialogs.downloadISOOnPremise.noticeText')" />
            <BaseRadioButtons :RadioGroup="isoRadioButtons" />
          </v-form>
          <template #actions>
            <BaseButton size="small" :text="$t('dashboard.buttons.cancel')" @click="cancelDownloadISO" />
            <BaseButton size="small" color="primary" :text="$t('dashboard.buttons.download')" @click="confirmDownloadISO" :disabled="!isValid"/>
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
import { downloadISO } from '@/services/provisioning.service';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { reactive } from 'vue';

const { t } = useI18n();
const nodeStates = ref([]);

const actions = computed(() => [
  {
    text: t('dashboard.buttons.downloadISOOnPremise'),
    color: 'primary',
    onClick: () => dialogDownloadISOOnPremise.value = true
  }
]);

const isoRadioButtons = reactive(new RadioGroup({
  label: "ISO choice",
  precision: "Choose the architecture of the ISO you want to download",
  options: [
    {
      label: "ARM",
      value: 'arm',
    },
    {
      label: "AMD",
      value: 'amd',
    }
  ],
  required: true,
  rules: [(v) => !!v || t('dashboard.dialogs.downloadISOOnPremise.radioOptions.required') ]
}));

// State
const dialogDownloadISOOnPremise = ref(false);
const isValid = ref(false);

nodeStates.value = [
  { Nodename: 'node-1', role: 'Control-plane', state: 'Ready' },
  { Nodename: 'node-2', role: 'Worker', state: 'NotReady' },
];

// Methods
function cancelDownloadISO() {
    isoRadioButtons.value = undefined;
    dialogDownloadISOOnPremise.value = false;
}

function confirmDownloadISO() {
    if (!isValid.value) return;

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
</script>