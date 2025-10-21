export class TextField {
    constructor({ label, value, type, required, rules, disabled, min, max, readonly }) {
        if (type === 'date' && !max) {
            max = '2099-12-31';
        }
        this.label = label;
        this.value = value;
        this.type = type;
        this.min = min;
        this.max = max;
        this.required = required;
        this.readonly = readonly;
        this.rules = rules;
        this.disabled = disabled;
    }

    getLabel() {
        return this.label;
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

    getType() {
        return this.type;
    }

    getMin() {
        return this.min;
    }

    getMax() {
        return this.max;
    }

    isDisabled() {
        return this.disabled;
    }

    setLabel(label) {
        this.label = label;
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

    setType(type) {
        this.type = type;
    }

    setMin(min) {
        this.min = min;
    }

    setMax(max) {
        this.max = max;
    }

    change(value) {
        this.value = value;
    }
}
