<template>
    <UILayout>
        <BaseLabelBar :title="$t('ui.tables.title')" :subheading="$t('ui.tables.subheading')" />
        <v-row>
            <v-col cols="12">
                <ComponentPreview :title="$t('ui.tables.title')" :code="baseDataTableCode" fullWidth>
                    <BaseDataTable v-model="effectiveSearch" :headers="headers" :items="items" :loading="loading"
                        :loadingText="$t('ui.tables.dataTables.loadingText')"
                        :noDataText="$t('ui.tables.dataTables.noDataText')"
                        :itemsPerPageText="$t('ui.tables.dataTables.itemsPerPageText')"
                        :titleToolbar="$t('ui.tables.dataTables.titleToolbar')"
                        :actionsButtonForTable="actionsButtonForTable" :rowClickable="fields.rowClickable.value">
                        <template #[`item.labels`]="{ item }">
                            <v-chip v-for="(label, index) in item.labels" :key="index" class="ma-1">
                                {{ label }}
                            </v-chip>
                        </template>
                    </BaseDataTable>
                    <template #controls>
                        <BaseCheckbox :Checkbox="fields.searchable" />
                        <BaseCheckbox :Checkbox="fields.rowClickable" />
                    </template>
                </ComponentPreview>
            </v-col>
        </v-row>
    </UILayout>
</template>

<script setup>
import { Checkbox } from '@/models/Checkbox'
import { computed, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();
const loading = ref(false);
const search = ref('');

const headers = [
    { title: 'Name', value: 'name' },
    { title: 'Role', value: 'role', align: "center" },
    { title: 'Status', value: 'status', align: "center" },
    { title: 'Labels', value: 'labels', align: "center" },
];
const items = [
    { name: 'Node 1', role: 'Master', status: 'Active', labels: ['test1', 'test2'] },
    { name: 'Node 2', role: 'Worker', status: 'Inactive', labels: ['test3'] },
    { name: 'Node 3', role: 'Worker', status: 'Active', labels: ['test4', 'test5'] },
];

const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: () => { }
    }
]);
const effectiveSearch = computed({
    get: () => (fields.searchable.value ? search.value : undefined),
    set: (val) => {
        if (fields.searchable.value) {
            search.value = val;
        }
    }
});
const baseDataTableCode = computed(() =>
    `
<BaseDataTable
    ${fields.searchable.value ? 'v-model="effectiveSearch"' : ''}
    :headers="headers"
    :items="items"
    :loading="loading"
    :loadingText="$t('ui.tables.dataTables.loadingText')"
    :noDataText="$t('ui.tables.dataTables.noDataText')"
    :itemsPerPageText="$t('ui.tables.dataTables.itemsPerPageText')"
    :titleToolbar="$t('ui.tables.dataTables.titleToolbar')"
    :actionsButtonForTable="actionsButtonForTable"
   ${fields.rowClickable.value ? 'rowClickable' : ''}
>
    <template #[\`item.labels\`]="{ item }">
        <v-chip v-for="(label, index) in item.labels" :key="index" class="ma-1">
            {{ label }}
        </v-chip>
    </template>
</BaseDataTable>
    `
);

const fields = reactive({
    searchable: new Checkbox({
        label: t('ui.tables.controls.searchable.label')
    }),
    rowClickable: new Checkbox({
        label: t('ui.tables.controls.rowClickable.label')
    }),
});
</script>