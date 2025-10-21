<template>
    <PortalLayout>
        <BaseLabelBar :title="$t('administration.teams.title')" :subheading="$t('administration.teams.subheading')" />
        <v-sheet class="mt-4 border rounded">
            <v-data-table 
                :headers="teamHeaders" 
                :items="teams" 
                :items-length="teams.length" 
                :loading=loading
                :search="search" :loading-text="$t('administration.teams.table.loadingText')"
                :no-data-text="$t('administration.teams.table.noDataText')" :items-per-page="10"
                :items-per-page-text="$t('administration.teams.table.itemsPerPageText')" 
                mobile-breakpoint="md" :hide-default-footer="teams.length < 10"
                @click:row="(event, item) => {
                    if (event.target.closest('.v-btn, .v-icon')) return
                    showViewDetailsDialog(item.item)
                }"
            >
                <template v-slot:top>
                    <BaseToolbarTable :title="$t('administration.teams.table.title')" :buttons="actionsButtonForTable" />
                    <v-text-field v-model="search" label="Search" prepend-inner-icon="mdi-magnify" variant="outlined"
                        hide-details single-line dense class="pa-3" />
                </template>
                <template #[`item.actions`]="{ item }">
                    <v-btn v-tooltip="{ text: $t('administration.teams.buttons.addUserToTeam') }" icon="mdi-account-plus" size="small" variant="text" @click="showAddUserToTeamDialog(item)" />
                    <v-btn v-tooltip="{ text: $t('administration.teams.buttons.deleteTeam') }" icon="mdi-delete" size="small" variant="text" @click="deleteTeam(item)" />
                </template>
            </v-data-table>
        </v-sheet>
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

<style scoped>
:deep(.v-data-table__tr:hover) {
  background-color: rgba(var(--v-theme-on-surface), 0.1);
}
</style>