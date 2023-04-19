<template>
  <div>
    <b-container fluid>
      <b-row class="p-1">
        <b-col class="mt-4 align-top">
          <b-row>
            <b-col>
              <node-grid></node-grid>
            </b-col>
          </b-row>
          <b-row>
            <b-col>
              <reservation-table></reservation-table>
            </b-col>
          </b-row>
        </b-col>
        <b-col class="mt-4  mr-2 p-1 align-top" rowspan="2">
          <tab-menu></tab-menu>
        </b-col>
      </b-row>
    </b-container>
  </div>
</template>

<script>
import NodeGrid from "./NodeGrid.vue";
import ReservationTable from "./ReservationTable.vue";
import TabMenu from "./TabMenu.vue";
import axios, { Axios } from "axios";
import moment from "moment";
export default {
  components: { NodeGrid, ReservationTable, TabMenu },
  name: "UserView",
  props: {
    msg: String,
  },
  data() {
    return {
      hosts: [],
      hostNames: [],
      reservations: [],
      clusterName: "",
      clusterPrefix: "",
      hostsPowered: [],
      hostsDown: [],
      hostsUnknown: [],
      hostsReserved: [],
      hostsOtherReserved: [],
      hostsGrpReserved: [],
      hostsResvPow: [],
      hostsResvDown: [],
      hostsResvUnknown: [],
      hostsGrpResvPow: [],
      hostsGrpResvDown: [],
      hostsGrpResvUnknown: [],
      hostsOtherResvPow: [],
      hostsOtherResvDown: [],
      hostsOtherResvUnknown: [],
      hostsAvlDown: [],
      hostsAvlUnknown: [],
      hostsAvlPow: [],
      hostsBlockedDown: [],
      hostsBlockedUnknown: [],
      hostsBlockedPow: [],
      hostsBlocked: [],
      hostsInstErrDown: [],
      hostsInstErrUnknown: [],
      hostsInstErrPow: [],
      hostsInstErr: [],
      hostsRestrictedPow: [],
      hostsRestrictedDown: [],
      hostsRestrictedUnknown: [],
      hostsForResv: [],
      distros: [],
      profiles: [],
      activeProfiles: [],
      activeDistros: [],
      eDistroNames: [],
      eProfileNames: [],
      userReservations: [],
      users: [],
      futureResv: false,
    };
  },
  mounted() {
    let showAllUrl = this.$config.IGOR_API_BASE_URL;

    axios
      .get(showAllUrl, { withCredentials: true })
      .then((response) => {
        // User is authenticated, token is valid
        this.$store.dispatch("loggedIn", true);

        // Save Host Data
        this.$store.dispatch("insertHosts", response.data.data.show.hosts);

        // Save Clustername
        this.$store.dispatch(
          "insertClusterName",
          response.data.data.show.cluster.name
        );
        this.clusterName = response.data.data.show.cluster.name;

        // Save Cluster prefix
        this.$store.dispatch(
          "insertClusterPrefix",
          response.data.data.show.cluster.prefix
        );
        this.clusterPrefix = response.data.data.show.cluster.prefix;
      })
      .catch(function(error) {
        alert("Error: " + error.response.data.message);
      });
    this.getUsers(false);
    this.getUserGroups(false);
    this.serverData(false);
    this.fetchFromServer();
  },
  methods: {
    // Redirect to Login on Session expiration
    sessionExpired() {
      this.$store.dispatch("logout").then(() => {
        this.$router.push("/login");
        window.location.reload();
      });
    },

    saveActiveProfilesDistros() {
      this.reservations.forEach((element) => {
        this.activeDistros.push(element.distro);
        this.activeProfiles.push(element.profile);
      });
      this.activeDistros = [...new Set(this.activeDistros)];
      this.activeProfiles = [...new Set(this.activeProfiles)];
      this.$store.dispatch("insertActiveProfiles", this.activeProfiles);
      this.$store.dispatch("insertActiveDistros", this.activeDistros);
    },

    getUsers(refreshReq) {
      let options = {
        withCredentials: true,
        headers : { "X-Igor-Refresh" :  refreshReq.toString() }
      }
      let usersUrl = this.$config.IGOR_API_BASE_URL + "/users";
      axios
        .get(usersUrl, options)
        .then((response) => {
          // Save Users
          if (response.data.data.users) {
            let users = response.data.data.users;
            let userNames = [];
            users.forEach((element) => {
              this.users.push(element.name);
              userNames.push(element.name);
            });
            this.$store.dispatch("insertUsers", userNames);
            this.$store.dispatch("insertUserDetails", users);
          }
        })
        .catch((error) => {
          if (error.response.status === 401) {
            this.sessionExpired();
          } else {
            alert("Error: " + error.response.data.message);
          }
        });
    },

    getUserGroups(refreshReq) {
      let options = {
        withCredentials: true,
        headers : { "X-Igor-Refresh" :  refreshReq.toString() }
      }
      let groupUrl =
        this.$config.IGOR_API_BASE_URL + "/groups?showMembers=true";
      let ownerGroups = [];
      let ownerGroupNames = [];
      let memberGroupNames = [];
      let memberGroups = [];
      let groups = [];
      let groupNames = [];

      axios
        .get(groupUrl, options)
        .then((response) => {
          memberGroups = response.data.data.member;
          ownerGroups = response.data.data.owner;
          ownerGroups.forEach((group) => {
            groups.push(group);
            ownerGroupNames.push(group.name);
          });
          memberGroups.forEach((group) => {
            groups.push(group);
            memberGroupNames.push(group.name);
          });
          //Find the index of 'all' group for deletion
          const index = groups.findIndex((group) => group.name === "all");
          if (~index) {
            groups.splice(index, 1);
          }
          groupNames = groups.map((h) => h.name);
          this.$store.dispatch("insertGroups", groups);
          this.$store.dispatch("insertOwnerGroupNames", ownerGroupNames);
          this.$store.dispatch("insertMemberGroupNames", memberGroupNames);
          this.$store.dispatch("insertGroupNames", groupNames);
        })
        .catch(function(error) {
          if (error.response.status === 401) {
            this.sessionExpired();
          } else {
            alert("Error: " + error.response.data.message);
          }
        });
    },

    getHostStatus(allHosts) {
      this.getPoweredHosts(allHosts);
      this.getReservedHosts();
      this.getReservedHostStatus(allHosts);
    },

    getPoweredHosts(allHosts) {
      // let allHosts = this.$store.getters.hosts;
      allHosts.forEach((element) => {
        if (element.powered === "true") {
          this.hostsPowered.push(element.name);
        } else if (element.powered === "false") {
          this.hostsDown.push(element.name);
        } else if (element.powered === "unknown") {
          this.hostsUnknown.push(element.name);
        }
      });
      this.$store.dispatch("insertHostsPowered", this.hostsPowered);
      this.$store.dispatch("insertHostsDown", this.hostsDown);
      this.$store.dispatch("insertHostsUnknown", this.hostsUnknown);
    },

    getReservedHosts() {
      let allReservations = this.$store.getters.reservations;
      let user = sessionStorage.getItem("username");
      allReservations.forEach((element) => {
        if (element.start > moment().unix()) {
          this.futureResv = true;
        } else {
          this.futureResv = false;
        }
        if (!this.futureResv) {
          let owner = element.owner;
          if (owner === user) {
            let hosts = element.hosts;
            hosts.forEach((e) => {
              this.hostsReserved.push(e);
            });
          } else if (this.$store.getters.groupReservations.includes(element)) {
            let hosts = element.hosts;
            hosts.forEach((e) => {
              this.hostsGrpReserved.push(e);
            });
          } else {
            let hosts = element.hosts;
            hosts.forEach((e) => {
              this.hostsOtherReserved.push(e);
            });
          }
          element.hosts.forEach((h) => {
            const index = this.hostsForResv.findIndex((host) => host === h);
            this.hostsForResv.splice(index, 1);
          });
        }
      });
      this.$store.dispatch("insertHostsReserved", this.hostsReserved);
      this.$store.dispatch("insertHostsGrpReserved", this.hostsReserved);
      this.$store.dispatch("insertHostsOtherReserved", this.hostsOtherReserved);
      this.$store.dispatch("insertHostsForResv", this.hostsForResv);
    },

    getReservedHostStatus(allHosts) {
      allHosts.forEach((element) => {
        if (this.hostsReserved.includes(element.name)) {
          if (this.hostsPowered.includes(element.name)) {
            if (this.hostsInstErr.includes(element.name)) {
              this.hostsInstErrPow.push(element.name);
            } else {
              this.hostsResvPow.push(element.name);
            }
          } else if (this.hostsDown.includes(element.name)) {
            if (this.hostsInstErr.includes(element.name)) {
              this.hostsInstErrDown.push(element.name);
            } else {
              this.hostsResvDown.push(element.name);
            }
          } else {
            if (this.hostsInstErr.includes(element.name)) {
              this.hostsInstErrUnknown.push(element.name);
            } else {
              this.hostsResvUnknown.push(element.name);
            }
          }
        } else if (this.hostsOtherReserved.includes(element.name)) {
          if (this.hostsPowered.includes(element.name)) {
            this.hostsOtherResvPow.push(element.name);
          } else if (this.hostsDown.includes(element.name)) {
            this.hostsOtherResvDown.push(element.name);
          } else {
            this.hostsOtherResvUnknown.push(element.name);
          }
        } else if (this.hostsGrpReserved.includes(element.name)) {
          if (this.hostsPowered.includes(element.name)) {
            this.hostsGrpResvPow.push(element.name);
          } else if (this.hostsDown.includes(element.name)) {
            this.hostsGrpResvDown.push(element.name);
          } else {
            this.hostsGrpResvUnknown.push(element.name);
          }
        } else if (this.hostsBlocked.includes(element.name)) {
          if (this.hostsPowered.includes(element.name)) {
            this.hostsBlockedPow.push(element.name);
          } else if (this.hostsDown.includes(element.name)) {
            this.hostsBlockedDown.push(element.name);
          } else {
            this.hostsBlockedUnknown.push(element.name);
          }
        } else {
          if (this.hostsDown.includes(element.name)) {
            if (element.restricted) {
              this.hostsRestrictedDown.push(element.name);
            } else {
              this.hostsAvlDown.push(element.name);
            }
          } else if (this.hostsUnknown.includes(element.name)) {
            if (element.restricted) {
              this.hostsRestrictedUnknown.push(element.name);
            } else {
              this.hostsAvlUnknown.push(element.name);
            }
          } else if (this.hostsPowered.includes(element.name)) {
            if (element.restricted) {
              this.hostsRestrictedPow.push(element.name);
            } else {
              this.hostsAvlPow.push(element.name);
            }
          }
        }
      });

      this.$store.dispatch("insertHostsResvPow", this.hostsResvPow);
      this.$store.dispatch("insertHostsResvDown", this.hostsResvDown);
      this.$store.dispatch("insertHostsResvUnknown", this.hostsResvUnknown);
      this.$store.dispatch("insertHostsGrpResvPow", this.hostsGrpResvPow);
      this.$store.dispatch("insertHostsGrpResvDown", this.hostsGrpResvDown);
      this.$store.dispatch(
        "insertHostsGrpResvUnknown",
        this.hostsGrpResvUnknown
      );
      this.$store.dispatch("insertHostsOtherResvPow", this.hostsOtherResvPow);
      this.$store.dispatch("insertHostsOtherResvDown", this.hostsOtherResvDown);
      this.$store.dispatch(
        "insertHostsOtherResvUnknown",
        this.hostsOtherResvUnknown
      );
      this.$store.dispatch("insertHostsAvlDown", this.hostsAvlDown);
      this.$store.dispatch("insertHostsAvlUnknown", this.hostsAvlUnknown);
      this.$store.dispatch("insertHostsAvlPow", this.hostsAvlPow);
      this.$store.dispatch("insertHostsBlockedDown", this.hostsBlockedDown);
      this.$store.dispatch(
        "insertHostsBlockedUnknown",
        this.hostsBlockedUnknown
      );
      this.$store.dispatch("insertHostsBlockedPow", this.hostsBlockedPow);
      this.$store.dispatch("insertHostsInstErrDown", this.hostsInstErrDown);
      this.$store.dispatch(
        "insertHostsInstErrUnknown",
        this.hostsInstErrUnknown
      );
      this.$store.dispatch("insertHostsInstErrPow", this.hostsInstErrPow);
      this.$store.dispatch("insertHostsRestrictedPow", this.hostsRestrictedPow);
      this.$store.dispatch(
        "insertHostsRestrictedDown",
        this.hostsRestrictedDown
      );
      this.$store.dispatch(
        "insertHostsRestrictedUnknown",
        this.hostsRestrictedUnknown
      );
    },

    fetchFromServer() {
      setInterval(() => this.serverData(true), 5000);
    },

    clearDataState() {
      this.hostsPowered = [];
      this.hostsDown = [];
      this.hostsUnknown = [];
      this.hostsReserved = [];
      this.hostsGrpReserved = [];
      this.hostsOtherReserved = [];
      this.hostsResvPow = [];
      this.hostsResvDown = [];
      this.hostsResvUnknown = [];
      this.hostsGrpResvPow = [];
      this.hostsGrpResvDown = [];
      this.hostsGrpResvUnknown = [];
      this.hostsOtherResvPow = [];
      this.hostsOtherResvDown = [];
      this.hostsOtherResvUnknown = [];
      this.hostsAvlDown = [];
      this.hostsAvlUnknown = [];
      this.hostsAvlPow = [];
      this.hostsBlockedDown = [];
      this.hostsBlockedUnknown = [];
      this.hostsBlockedPow = [];
      this.hostsBlocked = [];
      this.hostsInstErrDown = [];
      this.hostsInstErrUnknown = [];
      this.hostsInstErrPow = [];
      this.hostsInstErr = [];
      this.hostsRestrictedPow = [];
      this.hostsRestrictedDown = [];
      this.hostsRestrictedUnknown = [];
      this.hostsForResv = [];
      this.distros = [];
      this.profiles = [];
      this.activeProfiles = [];
      this.activeDistros = [];
      this.eDistroNames = [];
      this.eProfileNames = [];
      this.userReservations = [];
      this.users = [];
    },

    serverData(refreshReq) {
      let options = {
        withCredentials: true,
        headers : { "X-Igor-Refresh" :  refreshReq.toString() }
      }
      this.clearDataState();
      this.getUsers(refreshReq);
      this.getUserGroups(refreshReq);
      let showUrl = this.$config.IGOR_API_BASE_URL;
      axios
        .get(showUrl, options)
        .then((response) => {
          // Fetching data that needs auto refersh frequently
          // Save server timezone
          this.$store.dispatch("saveServerTime", response.data.serverTime);

          this.hosts = response.data.data.show.hosts;
          this.hosts.forEach((element) => {
            this.hostsForResv.push(element.name);
          });

          // Save Reservation details
          if (response.data.data.show.reservations) {
            this.reservations = response.data.data.show.reservations;
            this.$store.dispatch(
              "insertReservations",
              response.data.data.show.reservations
            );
            this.$store.dispatch(
              "insertReservationsForFiltering",
              response.data.data.show.reservations.length
            );

            // Save Current Users Reservations
            let userReservations = [];
            response.data.data.show.reservations.forEach((element) => {
              if (element.owner === sessionStorage.getItem("username")) {
                userReservations.push(element);
              }
            });
            this.$store.dispatch("insertUserReservations", userReservations);

            //Save Group Reservations
            let groupReservations = [];
            response.data.data.show.reservations.forEach((element) => {
              if (this.$store.getters.groupNames.includes(element.group)) {
                groupReservations.push(element);
              }
            });
            this.$store.dispatch("insertGroupReservations", groupReservations);

            //Save All associated Reservations (owned and associated by group)
            let allAssociatedReservations = [];
            this.$store.getters.userReservations.forEach((element) => {
              allAssociatedReservations.push(element);
            });
            this.$store.getters.groupReservations.forEach((element) => {
              allAssociatedReservations.push(element);
            });
            this.$store.dispatch(
              "insertAssociatedReservations",
              allAssociatedReservations
            );

            // Save reserved hosts with installation error
            userReservations.forEach((element) => {
              if (element.installError != "") {
                this.hostsInstErr.push(element.name);
              }
            });
            this.$store.dispatch("insertHostsInstErr", this.hostsInstErr);
          }
          // Save Active Profiles and Distros
          this.saveActiveProfilesDistros();

          // Save Hosts
          let allHosts = response.data.data.show.hosts;
          allHosts.forEach((element, index) => {
            this.hostNames.push(element.name);
            if (element.state === "blocked") {
              this.hostsBlocked.push(element.name);
              const index = this.hostsForResv.findIndex(
                (host) => host === element.name
              );
              if (~index) {
                this.hostsForResv.splice(index, 1);
              }
            }
          });
          this.$store.dispatch("insertHostsBlocked", this.hostsBlocked);
          this.$store.dispatch("insertHostNames", this.hostNames);
          this.getHostStatus(allHosts);

          // Save Distros
          if (response.data.data.show.distros) {
            this.distros = response.data.data.show.distros;
            this.$store.dispatch("insertDistros", this.distros);
            this.eDistroNames = response.data.data.show.distros.map(
              (h) => h.name
            );
            this.$store.dispatch("insertEDistroNames", this.eDistroNames);
          }

          // Save Profile
          if (response.data.data.show.profiles) {
            this.profiles = response.data.data.show.profiles;
            this.$store.dispatch("insertProfiles", this.profiles);
            this.eProfileNames = response.data.data.show.profiles.map(
              (h) => h.name
            );
            this.$store.dispatch("insertEProfileNames", this.eProfileNames);
          }

          // Save motd
          this.$store.dispatch(
            "insertMotd",
            response.data.data.show.cluster.motd
          );
          this.$store.dispatch(
            "insertMotdFlag",
            response.data.data.show.cluster.motdUrgent
          );
        })
        .catch(function(error) {
          if (error.response.status === 401) {
            this.sessionExpired();
          } else {
            alert("Error: " + error.response.data.message);
          }
        });
    },
  },
};
</script>
