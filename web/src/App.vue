<template>
  <div id="app">
    <b-container fluid>
      <b-row>
        <b-col>
          <div class="m-2 border-bottom">
            <top-navigation></top-navigation>
          </div>
        </b-col>
      </b-row>
      <b-row>
        <b-col class="border-right align-top" cols="12" xl="auto">
          <side-menu></side-menu>
        </b-col>
        <b-col xl="11" cols="12">
          <router-view />
        </b-col>
      </b-row>
    </b-container>
  </div>
</template>

<script>
import TopNavigation from "./components/TopNavigation.vue";
import SideMenu from "./components/SideMenu.vue";

export default {
  components: { TopNavigation, SideMenu },
  name: "App",
};
</script>
