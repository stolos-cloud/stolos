<template>
    <div class="add-user-to-team-dialog">
        <BaseDialog v-model="isOpen" :title="$t('administration.teams.dialogs.addUserToTeam.title')" closable>
            <v-form v-model="isValidForm">
                <BaseAutoComplete :AutoComplete="formFields.userChoiceEmail" />
            </v-form>
            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
                <BaseButton :text="$t('actionButtons.add')" :disabled="!isValidForm" @click="addUserToTeam" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { AutoComplete } from "@/models/AutoComplete.js";
import { addUserIdToTeam } from "@/services/teams.service";
import { getUsers } from "@/services/users.service";
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";

const { t } = useI18n();
const { autoCompleteRules } = FormValidationRules();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    team: {
        type: Object
    }
});

// State
const isValidForm = ref(false);
const isOpen = ref(props.modelValue);
const usersData = ref([]);

// Form state
const formFields = reactive({
    userChoiceEmail: new AutoComplete({
        label: t('administration.teams.formfields.emailList'),
        items: usersData,
        noDataText: t('administration.teams.formfields.noDataText'),
        required: true,
        rules: autoCompleteRules
    }),
});

// Emits
const emit = defineEmits(['update:modelValue', 'userAdded']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => {
    emit('update:modelValue', val);
    if(val && props.team) {
        filterUsersNotInTeam();
    }
});

// Methods
function closeDialog() {
    formFields.userChoiceEmail.value = undefined;
    emit('update:modelValue', false);
}
function filterUsersNotInTeam() {
    getUsers()
    .then((response) => {
        const teamUserIds = props.team?.users?.map(user => user.id) || [];
        usersData.value = response.users
            .filter(user => !teamUserIds.includes(user.id) && user.role !== 'admin')
            .map(user => ({ label: user.email, value: user.id }));        
    })
    .catch((error) => {
        console.error("Error fetching users:", error);
    });
}
function addUserToTeam() {
    if (!isValidForm.value) return;
    showOverlay();

    const userId = formFields.userChoiceEmail.value;
    addUserIdToTeam(props.team.id, { user_id: userId })
    .then(() => {
        showNotification(t('administration.teams.notifications.addUserSuccess'), 'success');
        emit('userAdded');
    })
    .catch((error) => {
        console.error("Error adding user to team:", error);
    })
    .finally(() => {
        closeDialog();
        hideOverlay();
    });
}
</script>