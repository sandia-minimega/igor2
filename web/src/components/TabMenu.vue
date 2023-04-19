<template>
  <div>
    <b-card no-body>
      <b-tabs
        card
        active-nav-item-class="font-weight-bold text-dark"
        active-tab-class="text-dark"
        title-item-class="text-dark"
      >
        <b-tab active no-body>
          <template #title>
            <b-icon icon="house-fill" style="color: grey;" class="p-0"></b-icon>
          </template>
          <div>
            <home-tab></home-tab>
          </div>
        </b-tab>
        <b-tab title="Reservation" title-link-class="text-dark">
          <b-card-text><create-reservation></create-reservation></b-card-text>
        </b-tab>
        <b-tab title="Distro" title-link-class="text-dark">
          <b-card-text><create-distro></create-distro></b-card-text>
        </b-tab>
        <b-tab title="Profile" title-link-class="text-dark">
          <b-card-text><create-profile></create-profile></b-card-text>
        </b-tab>
        <b-tab title="Groups" title-link-class="text-dark">
          <b-card-text><create-group></create-group></b-card-text>
        </b-tab>
      </b-tabs>
    </b-card>
  </div>
</template>

<script>
import CreateDistro from "./CreateDistro.vue";
import CreateReservation from "./CreateReservation.vue";
import CreateGroup from "./CreateGroup.vue";
import CreateProfile from "./CreateProfile.vue";
import HomeTab from "./HomeTab.vue";
export default {
  components: {
    HomeTab,
    CreateDistro,
    CreateReservation,
    CreateGroup,
    CreateProfile,
  },
  name: "MainMenu",
  props: {
    msg: String,
  },
};
</script>
