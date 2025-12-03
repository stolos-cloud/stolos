<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('administration.namespaces.title')" :subheading="$t('administration.namespaces.subheading')" />
        <BaseDataTable
            v-model="search"
            :headers="namespaceHeaders"
            :items="namespaces"
            :loading="loading"
            :loadingText="$t('administration.namespaces.table.loadingText')"
            :noDataText="$t('administration.namespaces.table.noDataText')"
            :itemsPerPageText="$t('administration.namespaces.table.itemsPerPageText')"
            :titleToolbar="$t('administration.namespaces.table.title')"
            :actionsButtonForTable="actionsButtonForTable"
            rowClickable
            @click:row="(event, item) => showViewDetailsDialog(item.item)"
        >
            <template #[`item.actions`]="{ item }">
                <v-btn v-tooltip="{ text: $t('administration.namespaces.buttons.addUserToNamespace') }" icon="mdi-account-plus" size="small" variant="plain" @click="showAddUserToNamespaceDialog(item)" />
                <v-btn v-tooltip="{ text: $t('administration.namespaces.buttons.deleteNamespace') }" icon="mdi-delete" size="small" variant="plain" @click="deleteNamespace(item)" />
            </template>
        </BaseDataTable>
        <CreateNamespaceDialog v-model="dialogCreateNamespace" @namespaceCreated="fetchNamespaces" />
        <AddUserToNamespaceDialog v-model="dialogAddUserToNamespace" :namespace="selectedNamespace" @userAdded="fetchNamespaces" />
        <ViewDetailsNamespaceDialog  v-model="dialogViewDetailsNamespace" :namespace="selectedNamespace" @userDeletedFromNamespace="fetchNamespaces" />
        <BaseConfirmDialog ref="confirmDialog" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { getNamespaces, deleteNamespaceById } from '@/services/namespaces.service';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import CreateNamespaceDialog from "@/pages/dialogs/administration/CreateNamespaceDialog.vue";
import AddUserToNamespaceDialog from "@/pages/dialogs/administration/AddUserToNamespaceDialog.vue";
import ViewDetailsNamespaceDialog from "@/pages/dialogs/administration/ViewDetailsNamespaceDialog.vue";

const { t } = useI18n();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();
const route = useRoute();

// State
const dialogCreateNamespace = ref(false);
const dialogAddUserToNamespace = ref(false);
const dialogViewDetailsNamespace = ref(false);
const loading = ref(false);
const namespaces = ref([]);
const search = ref('');
const confirmDialog = ref(null);
const selectedNamespace = ref(null);

// Computed
const namespaceHeaders = computed(() => [
    { title: t('administration.namespaces.table.headers.namespaces'), value: 'name' },
    { title: t('administration.namespaces.table.headers.numberOfMembers'), value: 'numberOfUsers', sortable: false, align: 'center' },
    { title: t('administration.namespaces.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: fetchNamespaces
    },
    {
        icon: "mdi-plus",
        tooltip: t('administration.namespaces.buttons.createNewNamespace'),
        text: t('administration.namespaces.buttons.createNewNamespace'),
        click: showCreateNamespaceDialog
    }
]);

//Mounted
onMounted(() => {
    search.value = route.query.search || '';
    fetchNamespaces();
});

// Methods
function showCreateNamespaceDialog() {
    dialogCreateNamespace.value = true;
}
function showAddUserToNamespaceDialog(item) {
    selectedNamespace.value = item;
    dialogAddUserToNamespace.value = true;
}
function showViewDetailsDialog(item) {
    selectedNamespace.value = item;
    dialogViewDetailsNamespace.value = true;
}
function fetchNamespaces() {
    loading.value = true;
    getNamespaces().then(response => {        
        namespaces.value = response.namespaces
            .filter(namespace => namespace.name !== "administrators")
            .map(namespace => ({
                ...namespace,
                numberOfUsers: namespace.users?.length || 0
            }));
    }).catch(error => {
        console.error("Error fetching namespaces:", error);
    }).finally(() => {
        loading.value = false;
    });
}
function deleteNamespace(namespace) {
    confirmDialog.value.open({
        title: t('administration.namespaces.dialogs.deleteNamespace.title'),
        message: t('administration.namespaces.dialogs.deleteNamespace.confirmationText', { namespaceName: namespace.name }),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            deleteNamespaceConfirmed(namespace);
        }
    })
}
function deleteNamespaceConfirmed(namespace) {
    showOverlay();

    deleteNamespaceById(namespace.id)
    .then(() => {
        showNotification(t('administration.namespaces.notifications.deleteNamespaceSuccess'), 'success');
        fetchNamespaces();
    })
    .catch((error) => {
        console.error("Error deleting namespace:", error);
        showNotification(t('administration.namespaces.notifications.deleteNamespaceError'), 'error');
    })
    .finally(() => {
        hideOverlay();
    });
}
</script>
