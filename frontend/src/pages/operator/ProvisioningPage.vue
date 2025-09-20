<template>
    <PortalLayout>
        <BaseLabelBar 
            :title="$t('provisioning.title')"
            :subheading="$t('provisioning.subheading')"
            :actions="actions"
        />
        <BaseButton color="primary" @click="showDialog">{{ $t('provisioning.openDialogButton') }}</BaseButton>
        <BaseDialog 
            v-model="dialogTest"
            :title="$t('provisioning.dialogTitle')"
            width="600"
            closable
        >
            <template #actions>
                <v-btn text @click="handleCancel">{{ $t('provisioning.cancelText') }}</v-btn>
                <v-btn color="primary" @click="handleConfirm">{{ $t('provisioning.confirmText') }}</v-btn>
            </template>
        </BaseDialog>
    </PortalLayout>
</template>

<script setup>
import PortalLayout from '@/components/layouts/PortalLayout.vue';
import BaseLabelBar from '@/components/base/BaseLabelBar.vue';
import BaseDialog from '@/components/base/BaseDialog.vue';
import BaseButton from '@/components/base/BaseButton.vue';

import { ref } from 'vue';
const dialogTest = ref(false);
const actions = [
  {
    text: "testing",
    color: 'primary',
    //flat" | "text" | "outlined" | "elevated" | "tonal" | "plain
    variant: 'elevated',
    onClick: () => alert('Add new clicked!')
  }
];
function handleCancel() {
    dialogTest.value = false;
    alert('Cancel clicked');
}
function handleConfirm() {
    dialogTest.value = false;
    alert('Confirm clicked');
}
function showDialog() {
    dialogTest.value = true;
}

</script>