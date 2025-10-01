import { createStore } from "vuex";
import { user } from './modules/user.store';

export default createStore({
    modules: {
        user
    }
});