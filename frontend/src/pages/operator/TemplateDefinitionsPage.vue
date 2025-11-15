<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('templateDefinitions.title')" :subheading="$t('templateDefinitions.subheading')" />
        <BaseDataTable v-model="search" :headers="templateHeaders" :items="templates" :loading="loading"
            :loadingText="$t('templateDefinitions.table.loadingText')"
            :noDataText="$t('templateDefinitions.table.noDataText')"
            :itemsPerPageText="$t('templateDefinitions.table.itemsPerPageText')"
            :titleToolbar="$t('templateDefinitions.table.title')"
            :footerMessage="$t('templateDefinitions.table.footerMessage', { group: 'stolos.cloud' })"
            :actionsButtonForTable="actionsButtonForTable" rowClickable
            @click:row="(event, item) => showViewDetailsTemplateDialog(item.item)">
        </BaseDataTable>
        <CreateTemplateDialog v-model="dialogCreateTemplate" @templateCreated="fetchTemplates" />
        <ViewDetailsTemplateDialog v-model="dialogViewDetailsTemplate" :team="selectedTemplate" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getTeams } from '@/services/teams.service';
import CreateTemplateDialog from "./dialogs/templates/CreateTemplateDialog.vue";
import ViewDetailsTemplateDialog from "./dialogs/templates/ViewDetailsTemplateDialog.vue";

const { t } = useI18n();

// State
const loading = ref(false);
const search = ref('');
const dialogCreateTemplate = ref(false);
const dialogViewDetailsTemplate = ref(false);
const templates = ref([]);
const selectedTemplate = ref(null);

// Computed
const templateHeaders = computed(() => [
    { title: t('templateDefinitions.table.headers.name'), value: 'name' },
    { title: t('templateDefinitions.table.headers.version'), value: 'version', sortable: false, align: 'center' },
    { title: t('templateDefinitions.table.headers.metadata'), value: 'metadata', sortable: false, align: 'center' },
    { title: t('templateDefinitions.table.headers.deployedApps'), value: 'deployedApps', sortable: false, align: 'center' }, //TODO : A voir ici si cest les bons noms de propriétés
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-text-box",
        tooltip: t('actionButtons.viewDocs'),
        text: t('actionButtons.viewDocs'),
        click: () => console.log('Docs button clicked')
    },
    {
        icon: "mdi-plus",
        tooltip: t('templateDefinitions.buttons.createNewTemplate'),
        text: t('templateDefinitions.buttons.createNewTemplate'),
        click: showCreateTemplateDialog
    }
]);

//Mounted
onMounted(() => {
    fetchTemplates();
});

// Methods
function showCreateTemplateDialog() {
    dialogCreateTemplate.value = true;
}
function showViewDetailsTemplateDialog(template) {
    selectedTemplate.value = template;
    dialogViewDetailsTemplate.value = true;
}
function fetchTemplates() {
    templates.value = [{
        name: "name1",
        version: "1.0",
        metadata: "",
        deployedApps: "3"
    },
    {
        name: "name2",
        version: "1.0",
        metadata: "",
        deployedApps: "2"
    }];

}
</script>