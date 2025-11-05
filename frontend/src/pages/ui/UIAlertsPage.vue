<template>
    <UILayout>
        <BaseLabelBar :title="$t('ui.alerts.title')" :subheading="$t('ui.alerts.subheading')" />
        <v-row>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.alerts.notice.title')" :code="noticeCode">
                    <BaseNotice :title="fields.title.value ? 'Test' : ''" text="Test"
                        :type="fields.noticeType.value" :density="fields.noticeDensity.value"
                        :closable="fields.closable.value">
                    </BaseNotice>
                    <template #controls>
                        <BaseSelect v-model="fields.noticeType.value" :Select="fields.noticeType" />
                        <BaseSelect v-model="fields.noticeDensity.value" :Select="fields.noticeDensity" />
                        <BaseCheckbox :Checkbox="fields.title" />
                        <BaseCheckbox :Checkbox="fields.closable" />
                    </template>
                </ComponentPreview>
            </v-col>
            <v-col cols="12" md="6" lg="6">
                <ComponentPreview :title="$t('ui.alerts.snackbar.title')" :code="snackbarCode">
                    <BaseButton :text="$t('ui.alerts.snackbar.button.showSnackbar')" @click="showNotification" />
                    <BaseNotification v-model="notif.visible" :text="notif.text" :type="fields.snackbarType.value" />
                    <template #controls>
                        <BaseSelect v-model="fields.snackbarType.value" :Select="fields.snackbarType" />
                    </template>
                </ComponentPreview>
            </v-col>
        </v-row>
    </UILayout>
</template>

<script setup>
import { Checkbox } from '@/models/Checkbox'
import { Select } from '@/models/Select'
import { computed, reactive, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

// Computed
const noticeCode = computed(() => `<BaseNotice ${fields.title.value ? `title="test"` : ''} type="${fields.noticeType.value}" density="${fields.noticeDensity.value}" ${fields.closable.value ? 'closable' : ''}>`);
const snackbarCode = computed(() => `<BaseNotification v-model="notif.visible" text="Test" type="${fields.snackbarType.value}" />`);

// Reactive form fields
const fields = reactive({
    noticeType: new Select({
        label: t('ui.alerts.notice.type.label'),
        options: [
            { label: t('ui.alerts.notice.type.options.success'), value: 'success' },
            { label: t('ui.alerts.notice.type.options.info'), value: 'info' },
            { label: t('ui.alerts.notice.type.options.warning'), value: 'warning' },
            { label: t('ui.alerts.notice.type.options.error'), value: 'error' }
        ],
    }),
    noticeDensity: new Select({
        label: t('ui.alerts.notice.density.label'),
        options: [
            { label: t('ui.alerts.notice.density.options.default'), value: 'default' },
            { label: t('ui.alerts.notice.density.options.compact'), value: 'compact' },
            { label: t('ui.alerts.notice.density.options.comfortable'), value: 'comfortable' }
        ],
    }),
    title: new Checkbox({
        label: t('ui.alerts.notice.controls.title.label')
    }),
    closable: new Checkbox({
        label: t('ui.alerts.notice.controls.closable.label')
    }),
    snackbarType: new Select({
        label: t('ui.alerts.snackbar.type.label'),
        options: [
            { label: t('ui.alerts.snackbar.type.options.success'), value: 'success' },
            { label: t('ui.alerts.snackbar.type.options.info'), value: 'info' },
            { label: t('ui.alerts.snackbar.type.options.warning'), value: 'warning' },
            { label: t('ui.alerts.snackbar.type.options.error'), value: 'error' }
        ],
    }),
});
const notif = reactive({
    visible: false,
    text: 'Test'
});

// onMounted
onMounted(() => {
    fields.noticeType.value = 'success';
    fields.noticeDensity.value = 'default';
    fields.snackbarType.value = 'success';
});

// Methods
function showNotification() {
    notif.type = fields.noticeType.value;
    notif.visible = true;
}
</script>