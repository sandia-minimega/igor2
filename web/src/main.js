import Vue from "vue";
import App from "./App.vue";
import Router from "vue-router";
import Vuex, { Store } from "vuex";
import store from "./store.js";
import router from "./router.js";
import Axios from "axios";
import TopNavigation from "./components/TopNavigation.vue";

import { BootstrapVue, BootstrapVueIcons } from "bootstrap-vue";
import "bootstrap/dist/css/bootstrap.css";
import "bootstrap-vue/dist/bootstrap-vue.css";
import "./assets/style/general-style.css";
import "./assets/style/nodegrid-style.css";
import "./assets/style/style.css";

Vue.prototype.$http = Axios;

Vue.prototype.$tagUser = false;
const token = localStorage.getItem("token");
if (token) {
  Vue.prototype.$http.defaults.headers.common["Authorization"] = token;
}
Vue.use(BootstrapVue);
Vue.use(BootstrapVueIcons);
Vue.component("top-navigation", TopNavigation);
Vue.config.productionTip = false;

Vue.use(Vuex);
Vue.use(Router);
Vue.use(router);
fetch(process.env.BASE_URL + "config.json")
  .then((response) => response.json())
  .then((config) => {
    Vue.prototype.$config = config;
    new Vue({
      router,
      store,
      render: (h) => h(App),
    }).$mount("#app");
  });

import VueRouter from "vue-router";
Vue.use(VueRouter);

// Handle navigation duplication for router push (Globally)

const originalPush = VueRouter.prototype.push;
VueRouter.prototype.push = function push(location) {
  return originalPush.call(this, location).catch((error) => {});
};
