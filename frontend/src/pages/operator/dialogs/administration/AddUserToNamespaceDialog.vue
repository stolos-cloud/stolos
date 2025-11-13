<template>
    <div class="add-user-to-namespace-dialog">
        <BaseDialog v-model="isOpen" :title="$t('administration.namespaces.dialogs.addUserToNamespace.title')" closable>
            <v-form v-model="isValidForm">
                <BaseAutoComplete :AutoComplete="formFields.userChoiceEmail" />
            </v-form>
            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
                <BaseButton :text="$t('actionButtons.add')" :disabled="!isValidForm" @click="addUserToNamespace" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { AutoComplete } from "@/models/AutoComplete.js";
import { addUserIdToNamespace } from "@/services/namespaces.service";
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
    namespace: {
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
        label: t('administration.namespaces.formfields.emailList'),
        items: usersData,
        noDataText: t('administration.namespaces.formfields.noDataText'),
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
    if(val && props.namespace) {
        filterUsersNotInNamespace();
    }
});

// Methods
function closeDialog() {
    formFields.userChoiceEmail.value = undefined;
    emit('update:modelValue', false);
}
function filterUsersNotInNamespace() {
    getUsers()
    .then((response) => {
        const namespaceUserIds = props.namespace?.users?.map(user => user.id) || [];
        usersData.value = response.users
            .filter(user => !namespaceUserIds.includes(user.id) && user.role !== 'admin')
            .map(user => ({ label: user.email, value: user.id }));
    })
    .catch((error) => {
        console.error("Error fetching users:", error);
    });
}
function addUserToNamespace() {
    if (!isValidForm.value) return;
    showOverlay();

    const userId = formFields.userChoiceEmail.value;
    addUserIdToNamespace(props.namespace.id, { user_id: userId })
    .then(() => {
        showNotification(t('administration.namespaces.notifications.addUserSuccess'), 'success');
        emit('userAdded');
    })
    .catch((error) => {
        console.error("Error adding user to namespace:", error);
    })
    .finally(() => {
        closeDialog();
        hideOverlay();
    });
}
</script>
