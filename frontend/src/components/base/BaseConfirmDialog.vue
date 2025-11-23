<template>
    <BaseDialog v-model="isOpen" :title="title" closable>
        <template #default>
            <p>{{ message }}</p>
        </template>
        <template #actions>
            <BaseButton v-if="showCancelButton" size="small" variant="outlined" :text="$t('buttons.cancel')" @click="cancel" />
            <BaseButton size="small" :text="confirmText" @click="confirm" />
        </template>
    </BaseDialog>
</template>

<script setup>
import { ref } from 'vue';

const isOpen = ref(false);
const title = ref('');
const message = ref('');
const confirmText = ref('');
const showCancelButton = ref(false);

const agreeCallback = ref(null);
const cancelCallback = ref(null);

//Methods
function open({
    title: dialogTitle,
    message: dialogMessage,
    confirmText: dialogConfirmText = 'OK',
    showCancelButton: dialogShowCancelButton = true,
    onConfirm = null,
    onCancel = null
} = {}) {
    title.value = dialogTitle;
    message.value = dialogMessage;
    confirmText.value = dialogConfirmText;
    showCancelButton.value = !dialogShowCancelButton;
    agreeCallback.value = onConfirm;
    cancelCallback.value = showCancelButton.value ? onCancel : null;
    isOpen.value = true;
}
function close() {
    isOpen.value = false;
}
function confirm() {
    close();
    if (agreeCallback.value) {
        agreeCallback.value();
    }
}
function cancel() {
    close();
    if (cancelCallback.value) {
        cancelCallback.value();
    }
}

// Expose methods to parent component
defineExpose({
    open,
    close
});
</script>
