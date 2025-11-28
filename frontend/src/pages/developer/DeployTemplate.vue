<template>
    <div ref="container" class="border rounded" style="min-height: 300px;"></div>
</template>

<script setup lang="ts">
import * as monaco from "monaco-editor"
import { configureMonacoYaml } from 'monaco-yaml'
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker"
import JsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker"
import YamlWorker from 'monaco-yaml/yaml.worker?worker'
import { useStore } from 'vuex';
import { ref, useTemplateRef, onMounted, computed, onBeforeUnmount } from 'vue'
import { getTemplate } from '../../services/templates.service'

const props = defineProps({
    templateId: {
        type: String,
        required: true
    }
})

const store = useStore();

const template = ref(null);

// Computed 
const isDark = computed(() => store.getters['user/getTheme'] === "dark");

function yamlToSnippet(yamlText) {
    const lines = yamlText.split('\n');
    let index = 1;
    const re = /^(\s*)([^:]+):\s*(.*)$/;
    const excludedKeys = new Set(['apiVersion', 'kind']);
    const transformed = lines.map(line => {
        const m = line.match(re);
        if (!m) return line;
        const [, indent, key, val] = m;
        if (val === '') return line; // skip empty
        if (excludedKeys.has(key.trim())) return line; //skip excluded keys
        const placeholder = `\${${index++}:${val}}`;
        return `${indent}${key}: ${placeholder}`;
    });
    return transformed.join('\n');
}

const jsonWorker = new JsonWorker()
const yamlWorker = new YamlWorker()
const editorWorker = new EditorWorker()

self.MonacoEnvironment = {
    getWorker(_, label) {
        if (label === "json") {
            return jsonWorker
        }
        if (label === "yaml") {
            return yamlWorker
        }
        return editorWorker
    }
}

const container = useTemplateRef('container')
onMounted(async () => {
    console.log(container.value)
    template.value = await getTemplate(props.templateId)
    console.log(template)

    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({ schemas: [] });
    const modelUri = `file:///${template.value.name}.yaml`;

    configureMonacoYaml(monaco, {
        enableSchemaRequest: true,
        schemas: [
            {
                // If YAML file is opened matching this glob
                fileMatch: [modelUri],
                // The following schema will be applied
                schema: JSON.parse(JSON.stringify(template.value.jsonSchema)),
                // And the URI will be linked to as the source.
                uri: `https://stolos.cloud/schemas/${template.value.name}`
            }
        ],
        isKubernetes: true
    })
    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
        validate: true,
        allowComments: false,
        schemas: [
            {
                uri: 'https://stolos.cloud/my-schema.json', // can be any unique URI
                fileMatch: ['*'],
                schema: template.value.jsonSchema
            },
        ],
    })

    const model = monaco.editor.createModel(
        "",
        "yaml",
        monaco.Uri.parse(`file:///${template.value.name}.yaml`)
    )

    const editor = monaco.editor.create(container.value, {
        model: model,
        automaticLayout: true,
        quickSuggestions: {
            other: true,
            comments: false,
            strings: true
        },
        theme: isDark.value ? "vs-dark" : "vs-light",
    });

    const snippetController = editor.getContribution('snippetController2');
    editor.focus();
    snippetController.insert(yamlToSnippet(template.value.defaultYaml));

})
onBeforeUnmount(() => {
    const editor = monaco.editor.getEditors().find(ed => ed.getModel()?.uri.path === `/${template.value.name}.yaml`);
    const model = monaco.editor.getModel(monaco.Uri.parse(`file:///${template.value.name}.yaml`));

    if (editor) {
        editor.dispose();
    }
    if (model) {
        model.dispose();
    }
})

function getEditorContent() {
    const model = monaco.editor.getModel(monaco.Uri.parse(`file:///${template.value.name}.yaml`));
    if (model) {
        return model.getValue();
    }
    return null;
}

defineExpose({
    getEditorContent
});
</script>