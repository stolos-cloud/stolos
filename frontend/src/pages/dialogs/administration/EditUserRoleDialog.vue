<template>
    <div class="edit-user-role-dialog">
        <BaseDialog v-model="isOpen" :title="$t('administration.users.dialogs.editUserRole.title', { userName: userName })" closable>
            <v-form v-model="isValidForm">
                <BaseSelect v-model="formFields.role.value" :Select="formFields.role" />
            </v-form>
            <template #actions>
                <BaseButton variant="outlined" :text="$t('actionButtons.cancel')" @click="closeDialog" />
                <BaseButton :text="$t('actionButtons.edit')" :disabled="!isValidForm" @click="updateRole" />
            </template>
        </BaseDialog>
    </div>
</template>

<script setup>
import { ref, computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useStore } from "vuex";
import { FormValidationRules } from "@/composables/FormValidationRules.js";
import { Select } from "@/models/Select.js";
import { updateUserRole } from '@/services/users.service';
import { GlobalNotificationHandler } from "@/composables/GlobalNotificationHandler";
import { GlobalOverlayHandler } from "@/composables/GlobalOverlayHandler";

const { t } = useI18n();
const store = useStore();
const { textfieldRules } = FormValidationRules();
const { showNotification } = GlobalNotificationHandler();
const { showOverlay, hideOverlay } = GlobalOverlayHandler();

const props = defineProps({
    modelValue: {
        type: Boolean,
        required: true
    },
    userSelected: {
        type: Object
    }
});

// State
const isValidForm = ref(false);
const isOpen = ref(props.modelValue);

// Computed
const userRoles = computed(() => store.getters['referenceLists/getUserRoles']);
const userName = computed(() => props.userSelected ? props.userSelected.email : '');

// Form state
const formFields = reactive({
    role: new Select({
        label: computed(() => t('administration.users.formfields.role')),
        options: userRoles.value,
        required: true,
        rules: textfieldRules
    })
});

// Emits
const emit = defineEmits(['update:modelValue', 'userRoleUpdated', 'update:userSelected']);

// Watchers
watch(() => props.modelValue, val => isOpen.value = val);
watch(isOpen, val => emit('update:modelValue', val));

// Methods
function closeDialog() {
    formFields.role.value = undefined;
    emit('update:modelValue', false);
}
function updateRole() {
    if (!isValidForm.value) return;
    showOverlay();

    const userToUpdate = props.userSelected.value;
    updateUserRole(userToUpdate.id, formFields.role.value)
        .then(() => {
            showNotification(t('administration.users.notifications.updateSuccess'), 'success');
            emit('userRoleUpdated');
        })
        .catch((error) => {
            showNotification(t('administration.users.notifications.updateError'), 'error');
            console.error("Error updating user role:", error);
        })
        .finally(() => {
            closeDialog();
            hideOverlay();
            emit('update:userSelected', null);
        });
}
</script>