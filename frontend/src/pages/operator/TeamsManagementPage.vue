<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('administration.teams.title')" :subheading="$t('administration.teams.subheading')" />
        <BaseDataTable
            v-model="search"
            :headers="teamHeaders"
            :items="teams"
            :loading="loading"
            :loadingText="$t('administration.teams.table.loadingText')"
            :noDataText="$t('administration.teams.table.noDataText')"
            :itemsPerPageText="$t('administration.teams.table.itemsPerPageText')"
            :titleToolbar="$t('administration.teams.table.title')"
            :actionsButtonForTable="actionsButtonForTable"
            rowClickable
            @click:row="(event, item) => showViewDetailsDialog(item.item)"
        >
            <template #[`item.actions`]="{ item }">
                <v-btn v-tooltip="{ text: $t('administration.teams.buttons.addUserToTeam') }" icon="mdi-account-plus" size="small" variant="text" @click="showAddUserToTeamDialog(item)" />
                <v-btn v-tooltip="{ text: $t('administration.teams.buttons.deleteTeam') }" icon="mdi-delete" size="small" variant="text" @click="deleteTeam(item)" />
            </template>
        </BaseDataTable>
        <CreateTeamDialog v-model="dialogCreateTeam" @teamCreated="fetchTeams" />
        <AddUserToTeamDialog v-model="dialogAddUserToTeam" :team="selectedTeam" @userAdded="fetchTeams" />
        <ViewDetailsTeamDialog  v-model="dialogViewDetailsTeam" :team="selectedTeam" @userDeletedFromTeam="fetchTeams" />
        <BaseConfirmDialog ref="confirmDialog" />
    </PortalLayout>
</template>

<script setup>
import { ref, onMounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { getTeams, deleteTeamById } from '@/services/teams.service';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";
import CreateTeamDialog from "./dialogs/administration/CreateTeamDialog.vue";
import AddUserToTeamDialog from "./dialogs/administration/AddUserToTeamDialog.vue";
import ViewDetailsTeamDialog from "./dialogs/administration/ViewDetailsTeamDialog.vue";

const { t } = useI18n();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

// State
const dialogCreateTeam = ref(false);
const dialogAddUserToTeam = ref(false);
const dialogViewDetailsTeam = ref(false);
const loading = ref(false);
const teams = ref([]);
const search = ref('');
const confirmDialog = ref(null);
const selectedTeam = ref(null);

// Computed
const teamHeaders = computed(() => [
    { title: t('administration.teams.table.headers.teams'), value: 'name' },
    { title: t('administration.teams.table.headers.numberOfMembers'), value: 'numberOfUsers', sortable: false, align: 'center' },
    { title: t('administration.teams.table.headers.actions'), value: 'actions', sortable: false, align: 'center' }
]);
const actionsButtonForTable = computed(() => [
    {
        icon: "mdi-refresh",
        tooltip: t('actionButtons.refresh'),
        text: t('actionButtons.refresh'),
        click: fetchTeams
    },
    {
        icon: "mdi-plus",
        tooltip: t('administration.teams.buttons.createNewTeam'),
        text: t('administration.teams.buttons.createNewTeam'),
        click: showCreateTeamDialog
    }
]);

//Mounted
onMounted(() => {
    fetchTeams();
});

// Methods
function showCreateTeamDialog() {
    dialogCreateTeam.value = true;
}
function showAddUserToTeamDialog(item) {
    selectedTeam.value = item;
    dialogAddUserToTeam.value = true;
}
function showViewDetailsDialog(item) {
    selectedTeam.value = item;
    dialogViewDetailsTeam.value = true;
}
function fetchTeams() {
    getTeams().then(response => {
        teams.value = response.teams
            .filter(team => team.name !== "administrators")
            .map(team => ({
                ...team,
                numberOfUsers: team.users?.length || 0
            }));
    }).catch(error => {
        console.error("Error fetching teams:", error);
    });
}
function deleteTeam(team) {
    confirmDialog.value.open({
        title: t('administration.teams.dialogs.deleteTeam.title'),
        message: t('administration.teams.dialogs.deleteTeam.confirmationText', { teamName: team.name }),
        confirmText: t('actionButtons.confirm'),
        onConfirm: () => {
            deleteTeamConfirmed(team);
        }
    })
}
function deleteTeamConfirmed(team) {
    showOverlay();

    deleteTeamById(team.id)
    .then(() => {
        showNotification(t('administration.teams.notifications.deleteTeamSuccess'), 'success');
        fetchTeams();
    })
    .catch((error) => {
        console.error("Error deleting team:", error);
        showNotification(t('administration.teams.notifications.deleteTeamError'), 'error');
    })
    .finally(() => {
        hideOverlay();
    });
}
</script>