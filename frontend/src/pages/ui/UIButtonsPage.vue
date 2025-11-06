<template>
    <UILayout>
        <BaseLabelBar :title="$t('ui.buttons.title')" :subheading="$t('ui.buttons.subheading')" />
        <v-row>
            <v-col v-for="example in buttonExamples" :key="example.id" cols="12" md="6" lg="6">
                <ComponentPreview :title="example.title" :code="example.code">
                    <BaseButton v-bind="example.buttonProps" text="Test" />
                    <template #controls>
                        <component v-for="control in example.controls" :key="control.id" :is="control.component"
                            v-bind="control.props" />
                    </template>
                </ComponentPreview>
            </v-col>
        </v-row>
    </UILayout>
</template>

<script setup>
import { Select } from '@/models/Select'
import { Checkbox } from '@/models/Checkbox'
import { computed, onMounted, reactive } from 'vue';
import { useI18n } from 'vue-i18n';
import BaseSelect from '@/components/base/BaseSelect.vue';
import BaseCheckbox from '@/components/base/BaseCheckbox.vue';

const { t } = useI18n();

// Reactive form fields
const formFields = reactive({
    sizes: new Select({
        label: t('ui.buttons.sizes.label'),
        options: [
            { label: t('ui.buttons.sizes.types.extraSmall'), value: 'x-small' },
            { label: t('ui.buttons.sizes.types.small'), value: 'small' },
            { label: t('ui.buttons.sizes.types.medium'), value: 'default' },
            { label: t('ui.buttons.sizes.types.large'), value: 'large' },
            { label: t('ui.buttons.sizes.types.extraLarge'), value: 'x-large' },
        ],
    }),
    variants: new Select({
        label: t('ui.buttons.variants.label'),
        options: [
            { label: t('ui.buttons.variants.types.flat'), value: 'flat' },
            { label: t('ui.buttons.variants.types.elevated'), value: 'elevated' },
            { label: t('ui.buttons.variants.types.tonal'), value: 'tonal' },
            { label: t('ui.buttons.variants.types.outlined'), value: 'outlined' },
            { label: t('ui.buttons.variants.types.text'), value: 'text' },
        ],
    }),
    icon: new Checkbox({
        label: t('ui.buttons.iconsTooltips.icons.label'),
    }),
    tooltip: new Checkbox({
        label: t('ui.buttons.iconsTooltips.tooltips.label'),
    }),
});

// onMounted
onMounted(() => {
    formFields.sizes.value = 'small';
    formFields.variants.value = 'flat';
});

// Computed
const buttonExamples = computed(() => [
    {
        id: 'size',
        title: t('ui.buttons.sizes.title'),
        code: `<BaseButton :size="${formFields.sizes.value}" text="Test" />`,
        buttonProps: {
            size: formFields.sizes.value
        },
        controls: [{
            id: 'size-select',
            component: BaseSelect,
            props: {
                modelValue: formFields.sizes.value,
                'onUpdate:modelValue': (v) => formFields.sizes.value = v,
                Select: formFields.sizes
            }
        }]
    },
    {
        id: 'variant',
        title: t('ui.buttons.variants.title'),
        code: `<BaseButton :variant="${formFields.variants.value}" text="Test" />`,
        buttonProps: {
            variant: formFields.variants.value
        },
        controls: [{
            id: 'variant-select',
            component: BaseSelect,
            props: {
                modelValue: formFields.variants.value,
                'onUpdate:modelValue': (v) => formFields.variants.value = v,
                Select: formFields.variants
            }
        }]
    },
    {
        id: 'icon-tooltip',
        title: t('ui.buttons.iconsTooltips.title'),
        code: computed(() => {
            let iconPart = formFields.icon.value ? ' :icon="\'mdi-check\'"' : '';
            let tooltipPart = formFields.tooltip.value ? ' :tooltip="\'Test\'"' : '';
            return `<BaseButton text="Test"${iconPart}${tooltipPart} />`
        }).value,
        buttonProps: {
            icon: formFields.icon.value ? 'mdi-check' : undefined,
            tooltip: formFields.tooltip.value ? 'Test' : undefined
        },
        controls: [
            {
                id: 'icon-checkbox',
                component: BaseCheckbox,
                props: { Checkbox: formFields.icon }
            },
            {
                id: 'tooltip-checkbox',
                component: BaseCheckbox,
                props: { Checkbox: formFields.tooltip }
            }
        ]
    }
]);
</script>
