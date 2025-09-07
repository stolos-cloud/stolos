import "vuetify/styles";
import { createVuetify } from "vuetify";
import * as components from "vuetify/components";
import * as directives from "vuetify/directives";

export default createVuetify({
    theme: {
      //TODO: Set good color scheme
        defaultTheme: "light",
    },
    components,
    directives,
});