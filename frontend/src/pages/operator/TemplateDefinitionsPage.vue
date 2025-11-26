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
            <template #[`item.deployedApps`]="{ item }">
                {{ item.deployedApps.length }}
            </template>
        </BaseDataTable>
        <CreateTemplateDialog v-model="dialogCreateTemplate" @templateCreated="fetchTemplates" />
        <ViewDetailsTemplateDialog v-model="dialogViewDetailsTemplate" :template="selectedTemplate" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getTemplates, listDeployments } from "@/services/templates.service";
import CreateTemplateDialog from "@/pages/dialogs/templates/CreateTemplateDialog.vue";
import ViewDetailsTemplateDialog from "@/pages/dialogs/templates/ViewDetailsTemplateDialog.vue";

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
    { title: t('templateDefinitions.table.headers.deployedApps'), value: 'deployedApps', sortable: false, align: 'center' }, 
    { title: t('templateDefinitions.table.headers.labels'), value: 'labels', sortable: false, width: "25%" },
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-text-box",
        tooltip: t('actionButtons.viewDocs'),
        text: t('actionButtons.viewDocs'),
        click: redirectToWikiDocs
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
async function fetchTemplates() {
    loading.value = true;
    getTemplates().then(async (response) => {
        const templatesWithDeployments = [];
        
        for (const tpl of response) {
            const deployments = await listDeployments({ template: tpl.name, namespace: '' });             
            templatesWithDeployments.push({
                ...tpl,
                deployedApps: deployments
            });
        }
        templates.value = templatesWithDeployments;

    }).catch(error => {
        console.error("Error fetching templates:", error);
    }).finally(() => {
        loading.value = false;
    });
}
function redirectToWikiDocs() {
    window.open("https://github.com/stolos-cloud/stolos/wiki");
}
</script>