import { createStore } from 'vuex';
import { user } from './modules/user.store';
import { referenceLists } from './modules/referenceLists.store';

export default createStore({
    modules: {
        user,
        referenceLists,
    },
});
