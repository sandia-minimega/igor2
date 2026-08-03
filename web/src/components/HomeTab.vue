<template>
  <div>
    <b-card no-body class="mt-3 mb-3">
      <b-tabs card active-nav-item-class="fw-bold">
        <b-tab active no-body title="My Reservations">
          <user-reservations embedded></user-reservations>
        </b-tab>
        <b-tab no-body title="My Profiles">
          <profile-table embedded></profile-table>
        </b-tab>
        <b-tab no-body title="My Distros">
          <distro-table embedded></distro-table>
        </b-tab>
      </b-tabs>
    </b-card>
  </div>
</template>

<script>
import ProfileTable from "./ProfileTable.vue";
import DistroTable from "./DistroTable.vue";
import UserReservations from "./UserReservations.vue";
export default {
  components: { ProfileTable, DistroTable, UserReservations },
  name: "HomeTab",
  props: {
    msg: String,
  },
};
</script>
