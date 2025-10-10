export class Checkbox {
    constructor({ label, value, readonly, required, rules }) {
        this.label = label;
        this.value = value;
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
