export class FileInput {
    constructor({ label, value, accept, readonly, required, rules }) {
        this.label = label;
        this.value = value;
        this.accept = accept;
        this.readonly = readonly;
        this.required = required;
        this.rules = rules;
    }

    getLabel() {
        return this.label;
    }

    getValue() {
        return this.value;
    }

    getAccept() {
        return this.accept;
    }

    isReadonly() {
        return this.readonly;
    }

    isRequired() {
        return this.required;
    }

    getRules() {
        return this.rules;
    }

    setLabel(label) {
        this.label = label;
    }

    setValue(value) {
        this.value = value;
    }

    setAccept(accept) {
        this.accept = accept;
    }

    setReadonly(readonly) {
        this.readonly = readonly;
    }

    setRequired(required) {
        this.required = required;
    }

    setRules(rules) {
        this.rules = rules;
    }

    change(value) {
        this.value = value;
    }
}
