export class TextField {
    constructor({ label, value, type, required, rules, disabled, minDate, maxDate, readonly }) {
        if (type === 'date' && !maxDate) {
            maxDate = "2099-12-31";
        }
        this.label = label;
        this.value = value;
        this.type = type;
        this.minDate = minDate;
        this.maxDate = maxDate;
        this.required = required;
        this.readonly=readonly;
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

    getMinDate() {
        return this.minDate;
    }

    getMaxDate() {
        return this.maxDate;
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

    setMinDate(minDate) {
        this.minDate = minDate;
    }

    setMaxDate(maxDate) {
        this.maxDate = maxDate;
    }  

    change(value) {
        this.value = value;
    }
}