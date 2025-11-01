<script setup lang="ts">

import * as monaco from "monaco-editor"
import { configureMonacoYaml } from 'monaco-yaml'
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker"
import JsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker"
import YamlWorker from 'monaco-yaml/yaml.worker?worker'
import { useTemplateRef, onMounted } from 'vue'
import { getTemplate } from '../../services/templates.service'
// import * as YAMLWorker from 'monaco-yaml/yaml.worker'

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
  const template = await getTemplate("stolosplatforms.stolos.cloud")
  console.log(template)

  configureMonacoYaml(monaco, {
    enableSchemaRequest: true,
    schemas: [
      {
        // If YAML file is opened matching this glob
        fileMatch: ['*'],
        // The following schema will be applied
        schema: template.jsonSchema,
        // And the URI will be linked to as the source.
        uri: `https://stolos.cloud/schemas/${template.name}`
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
        schema: template.jsonSchema
      },
    ],
  })

  const model = monaco.editor.createModel(
    "",
    undefined,
    monaco.Uri.parse(`file:///${template.name}.yaml`)
  )

  const editor = monaco.editor.create(container.value, {
    model: model,
    automaticLayout: true,
    quickSuggestions: {
      other: true,
      comments: false,
      strings: true
    }
  });

  const snippetController = editor.getContribution('snippetController2');
  editor.focus();
  snippetController.insert(yamlToSnippet(template.defaultYaml));

})


</script>

<template>
  <div ref="container" style="height: 500px"></div>
</template>

<style scoped>

</style>
