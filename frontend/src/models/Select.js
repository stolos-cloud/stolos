export class Select {
    constructor({ label, options, readonly, required, rules }) {
        this.label = label;
        this.options = options;
        this.readonly = readonly;
        this.required = required;
        this.rules = rules;
    }

    getLabel() {
        return this.label;
    }

    getOptions() {
        return this.options;
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

    setOptions(options) {
        this.options = options;
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
}
