import { useI18n } from 'vue-i18n';

export function FormValidationRules() {
    const { t } = useI18n();

    const emailRules = [
        v => !!v || t('rules.validation.email.required'),
        v => /.+@.+\..+/.test(v) || t('rules.validation.email.invalid'),
    ];
    const passwordRules = [
        v => !!v || t('rules.validation.password.required'),
        v => v.length >= 8 || t('rules.validation.password.invalid'),
        v => /[A-Z]/.test(v) || t('rules.validation.password.uppercase'),
        v => /[a-z]/.test(v) || t('rules.validation.password.lowercase'),
        v => /[0-9]/.test(v) || t('rules.validation.password.number'),
        v => /[!@#$%^&*(),.?":{}|<>-]/.test(v) || t('rules.validation.password.special'),
    ];
    const textfieldRules = [v => !!v || t('rules.validation.textfield.required')];
    const textfieldSlugRules = [
        v => !!v || t('rules.validation.textfield.required'),
        v => /^[^A-Z]*$/.test(v) || t('rules.validation.textfield.onlyLowercase'),
        v => !/[0-9]/.test(v) || t('rules.validation.textfield.noNumbers'),
        v => !/\s/.test(v) || t('rules.validation.textfield.noSpaces'),
    ];
    const autoCompleteRules = [
        v => (!!v && v.length > 0) || t('rules.validation.autoComplete.required'),
    ];

    return {
        emailRules,
        passwordRules,
        textfieldRules,
        textfieldSlugRules,
        autoCompleteRules,
    };
}
