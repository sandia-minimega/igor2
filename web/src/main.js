import { createApp } from "vue";
import App from "./App.vue";
import store, { setRuntimeConfig } from "./store.js";
import router from "./router.js";
import Axios from "axios";
import { BootstrapIconsPlugin } from "bootstrap-icons-vue";
import { BOffcanvas } from "bootstrap-vue-next/components/BOffcanvas";
import { BTab, BTabs } from "bootstrap-vue-next/components/BTabs";
import BIcon from "./components/BIcon.vue";

import "./assets/scss/bootstrap.scss";
import "bootstrap-vue-next/dist/bootstrap-vue-next.css";
import "./assets/scss/custom.scss";
import "./assets/scss/general-style.scss";

const THEME_STORAGE_KEY = "igorweb-theme";
if (localStorage.getItem(THEME_STORAGE_KEY) === "dark") {
    document.documentElement.setAttribute('data-bs-theme', "dark");
}

const token = localStorage.getItem("token");
if (token) {
  Axios.defaults.headers.common["Authorization"] = token;
}

fetch(process.env.BASE_URL + "config.json")
    .then((response) => response.json())
    .then((config) => {
        setRuntimeConfig(config);
        const app = createApp(App);

    app.config.globalProperties.$http = Axios;
    app.config.globalProperties.$tagUser = false;
    app.config.globalProperties.$config = config;

    // BootstrapVue 2 registered every icon globally.
    // BIcon allows backwards compatibility for now.
    app.use(BootstrapIconsPlugin);
    app.component("BIcon", BIcon);

    // These components are renamed or nested exports in BootstrapVueNext.
    // Register them explicitly so Vue CLI/webpack does not depend on resolver
    // behavior for the BootstrapVue 2 kebab-case component names.
    app.component("BOffcanvas", BOffcanvas);
    app.component("BTab", BTab);
    app.component("BTabs", BTabs);

    app.use(router);
    app.use(store);
    app.mount("#app");
  });
