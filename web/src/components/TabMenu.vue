<template>
  <div>
    <b-card no-body>
      <b-tabs
        card
        active-nav-item-class="fw-bold"
        active-tab-class=""
        title-item-class=""
      >
        <b-tab active no-body>
          <template #title>
            <b-icon icon="house-fill" style="color: grey;" class="p-0"></b-icon>
          </template>
          <div>
            <home-tab></home-tab>
          </div>
        </b-tab>
        <b-tab title="Reservation" >
          <b-card-text><create-reservation></create-reservation></b-card-text>
        </b-tab>
        <b-tab title="Distro" >
          <b-card-text><create-distro></create-distro></b-card-text>
        </b-tab>
        <b-tab title="Profile" >
          <b-card-text><create-profile></create-profile></b-card-text>
        </b-tab>
        <b-tab title="Groups" >
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
