<template>
    <PortalLayout>
        <BaseLabelBar
            :title="$t('dashboard.title')"
            :subheading="$t('dashboard.subheading')"
            :actions="actions"
        />
        <BaseDialog v-model="dialogDownloadISOOnPremise" :title="$t('dashboard.dialogs.downloadISOOnPremise.title')" width="600" closable>
          <v-form v-model="isValid">
            <BaseNotice type="info" :text="$t('dashboard.dialogs.downloadISOOnPremise.noticeText')" />
            <BaseRadioButtons :RadioGroup="isoRadioButtons" />
          </v-form>
          <template #actions>
            <BaseButton size="small" :text="$t('dashboard.buttons.cancel')" @click="handleCancel" />
            <BaseButton size="small" color="primary" :text="$t('dashboard.buttons.download')" @click="handleConfirm" :disabled="!isValid"/>
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
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { reactive } from 'vue';

const { t } = useI18n();
const actions = [
  {
    text: t('dashboard.buttons.downloadISOOnPremise'),
    color: 'primary',
    onClick: () => dialogDownloadISOOnPremise.value = true
  }
];

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

// Methods
function handleCancel() {
    isoRadioButtons.value = undefined;
    dialogDownloadISOOnPremise.value = false;
    
}
function handleConfirm() {
    dialogDownloadISOOnPremise.value = false;
}


</script>