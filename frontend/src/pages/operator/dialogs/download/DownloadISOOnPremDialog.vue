<template>
    <BaseDialog v-model="isOpen" :title="$t('dialogs.downloadISOOnPremise.title')" closable>
        <v-form v-model="isValidForm">
            <BaseNotice type="info" :text="$t('dialogs.downloadISOOnPremise.noticeText')" />
            <BaseRadioButtons :RadioGroup="isoRadioButtons" />
        </v-form>
        <template #actions>
            <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
            <BaseButton :text="$t('actionButtons.download')" :disabled="!isValidForm" @click="confirmDownloadISO" />
        </template>
    </BaseDialog>
</template>

<script setup>
import { ref, reactive, watch, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { RadioGroup } from '@/models/RadioGroup.js';
import { generateISO } from '@/services/provisioning.service';

const { t } = useI18n();
const store = useStore();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    namespace: {
        type: Object
    }
});

// State
const isValidForm = ref(false);
const isOpen = ref(props.modelValue);

// Computed
const listISOTypes = computed(() => store.getters['referenceLists/getIsoTypes']);

// Reactives
const isoRadioButtons = reactive(new RadioGroup({
    label: t('dialogs.downloadISOOnPremise.label'),
    precision: t('dialogs.downloadISOOnPremise.precision'),
    options: listISOTypes.value,
    required: true,
    rules: [(v) => !!v || t('rules.validation.radioGroup.required')]
}));

// Emits
const emit = defineEmits(['update:modelValue']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => {
    emit('update:modelValue', val);
});

// Methods
function closeDialog() {
    isoRadioButtons.value = undefined;
    emit('update:modelValue', false);
}
function confirmDownloadISO() {
    if (!isValidForm.value) return;

    generateISO({ architecture: isoRadioButtons.value })
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
            closeDialog();
        });
}
</script>
