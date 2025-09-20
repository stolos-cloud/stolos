<template>
  <AuthLayout>
    <v-container
    fluid
    class="fill-height d-flex justify-center align-center"
    >
      <v-card
        max-width="500"
        class="pa-6"
        elevation="8"
      >
        <v-card-title class="d-flex flex-column align-center">
          <v-img
            src="@/assets/logo.png"
            alt="Logo"
            max-width="80"
            class="mb-4"
          />
          <span class="text-h4 font-bold">Connexion</span>
        </v-card-title>

        <v-card-text>
          <v-form ref="loginForm" @submit.prevent="handleSubmit">
            <v-text-field
              label="Prénom"
              v-model="form.firstname"
              outlined
              dense
              required
              class="mb-4"
            />
            <v-text-field
              label="Nom de famille"
              v-model="form.lastname"
              outlined
              dense
              required
              class="mb-4"
            />
            <v-text-field
              label="Courriel"
              v-model="form.email"
              outlined
              dense
              required
              prepend-inner-icon="mdi-email-outline"
              class="mb-4"
            />
            <v-text-field
              label="Mot de passe"
              v-model="form.password"
              type="password"
              outlined
              dense
              required
              prepend-inner-icon="mdi-lock-outline"
              class="mb-4"
            />
            <v-text-field
              label="Confirmation de mot de passe"
              v-model="form.confirmPassword"
              type="password"
              outlined
              dense
              required
              prepend-inner-icon="mdi-lock-outline"
              class="mb-4"
            />

            <v-btn
              type="submit"
              color="primary"
              class="mt-4"
              block
            >
              Créer votre compte
            </v-btn>
          </v-form>
        </v-card-text>

        <v-card-actions class="justify-center mt-4">
          <span>Avez-vous déjà un compte ?</span>
          <RouterLink
            to="/"
            class="ml-2 text-decoration-none"
          >
            Connectez-vous
          </RouterLink>
        </v-card-actions>
      </v-card>
    </v-container>
  </AuthLayout>
</template>

<script setup>
import AuthLayout from "@/components/layouts/AuthLayout.vue";
import { reactive } from "vue";

const form = reactive({
  firstname: "",
  lastname: "",
  email: "",
  password: "",
  confirmPassword: "",
});

function handleSubmit() {
  console.log("Form submitted:", form);
}
</script>

<style scoped>
.text-decoration-none {
  text-decoration: none;
}
.text-decoration-none:hover {
  text-decoration: underline;
}
</style>
