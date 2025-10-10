export class RadioGroup {
    constructor({ label, precision, options, value, required, rules, disabled }) {
        this.label = label;
        this.precision = precision;
        this.options = options;
        this.value = value;
        this.required = required;
        this.rules = rules;
        this.disabled = disabled;
    }

    getLabel() {
        return this.label;
    }

    getPrecision() {
        return this.precision;
    }

    getOptions() {
        return this.options;
    }

    getValue() {
        return this.value;
    }

    isRequired() {
        return this.required;
    }

    getRules() {
        return this.rules;
    }

    isDisabled() {
        return this.disabled;
    }

    setLabel(label) {
        this.label = label;
    }

    setPrecision(precision) {
        this.precision = precision;
    }

    setOptions(options) {
        this.options = options;
    }

    setValue(value) {
        this.value = value;
    }

    setRequired(required) {
        this.required = required;
    }

    setRules(rules) {
        this.rules = rules;
    }

    setDisabled(disabled) {
        this.disabled = disabled;
    }

    change(value) {
        this.value = value;
    }
}
