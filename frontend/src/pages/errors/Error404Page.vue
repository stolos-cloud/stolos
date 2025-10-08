<template>
    <AuthLayout>
        <v-row d-flex align-content="center" justify="center" style="min-height: 100dvh;">
            <v-col cols="12" md="8" lg="6">
                <div class="d-flex align-center justify-center">
                    <v-icon size="150" color="error" class="me-3">mdi-alert-circle-outline</v-icon>
                    <div>
                        <BaseTitle class="mb-4" :level="1" :title="$t('errors.404.title')" />
                        <p class="mt-4">{{ $t('errors.404.description') }}</p>
                        <BaseButton
                            class="mt-4"
                            :text="$t('actionButtons.goHome')"
                            color="primary"
                            @click="$router.push({ name: 'login' })"
                        />
                    </div>
                </div>
            </v-col>
        </v-row>
    </AuthLayout>
</template>