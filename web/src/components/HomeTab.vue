<template>
  <div>
    <b-container fluid>
      <b-row class="mt-3">
        <b-col>
          <user-reservations></user-reservations>
        </b-col>
      </b-row>
      <b-row class="mt-3">
        <b-col>
          <profile-table></profile-table>
        </b-col>
      </b-row>
      <b-row class="mt-3 mb-3">
        <b-col>
          <distro-table></distro-table>
        </b-col>
      </b-row>
    </b-container>
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
