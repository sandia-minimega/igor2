import Vue from "vue";
import Vuex from "vuex";
import axios from "axios";

Vue.use(Vuex);

export default new Vuex.Store({
  state: {
    // Authentication
    status: "",
    username: "",
    isLoggedIn: false,

    // Application wide data
    serverTime: "",
    hosts: [],
    hostNames: [],
    reservations: [],
    reservationsFilteredLength: 0,
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
    hostsRestrictedDown: [],
    hostsRestrictedPow: [],
    hostsRestrictedUnknown: [],
    hostsForResv: [],
    selectedHosts: [],
    selectedHostsCount: 0,
    selectedHostID: [],
    hostSelectedPow: [],
    hostSelectedDown: [],
    hostSelectedUnknown: [],
    groups: [],
    groupNames: [],
    ownerGroupNames: [],
    memberGroupNames: [],
    distros: [],
    profiles: [],
    activeProfiles: [],
    activeDistros: [],
    eDistroNames: [],
    eProfileNames: [],
    userReservations: [],
    groupReservations: [],
    associatedReservations: [],
    users: [],
    userDetails: [],
    errors: [],
    motd: "",
    motdFlag: false,
    defaultReserveMinutes: "",
    vlanMin: "",
    vlanMax: "",
  },

  mutations: {
    // Authentication
    auth_request(state) {
      state.status = "loading";
    },
    auth_success(state, username) {
      state.status = "success";
      state.isLoggedIn = true;
      state.username = username;
    },
    auth_error(state) {
      state.status = "error";
    },
    logout(state) {
      state.status = "";
      state.username = "";
      state.isLoggedIn = false;
      sessionStorage.removeItem("username");
      sessionStorage.removeItem("authenticated");
    },
    LOGGED_IN(state, payload) {
      state.isLoggedIn = payload;
    },
    SAVE_SERVER_TIME(state, payload) {
      state.serverTime = payload;
    },
    // Hosts, Reservation, Profile, Distro
    INSERT_HOSTS(state, payload) {
      state.hosts = payload;
      state.hosts = [...new Set(state.hosts)];
    },
    INSERT_HOSTNAMES(state, payload) {
      state.hostNames = payload;
      state.hostNames = [...new Set(state.hostNames)];
      const sortAlphaNum = (a, b) =>
        a.localeCompare(b, "en", { numeric: true });
      state.hostNames.sort(sortAlphaNum);
    },
    INSERT_RESERVATIONS(state, payload) {
      state.reservations = payload;
      state.reservations = [...new Set(state.reservations)];
    },
    INSERT_RESERVATIONS_FOR_FILTERING(state, payload) {
      state.reservationsFilteredLength = payload;
    },
    INSERT_NEWRESERVATIONS(state, payload) {
      state.reservations.push(payload);
      state.reservations = [...new Set(state.reservations)];
    },
    INSERT_NEWPROFILE(state, payload) {
      state.profiles.push(payload);
      state.profiles = [...new Set(state.profiles)];
    },
    INSERT_NEWDISTRO(state, payload) {
      state.distros.push(payload);
      state.distros = [...new Set(state.distros)];
    },
    INSERT_NEWEDISTRONAMES(state, payload) {
      state.eDistroNames.push(payload);
      state.eDistroNames = [...new Set(state.eDistroNames)];
    },
    INSERT_ACTIVE_DISTROS(state, payload) {
      state.activeDistros.push(payload);
      state.activeDistros = [...new Set(state.activeDistros)];
    },
    INSERT_ACTIVE_PROFILES(state, payload) {
      state.activeProfiles.push(payload);
      state.activeProfiles = [...new Set(state.activeProfiles)];
    },
    INSERT_NEWGROUP(state, payload) {
      state.groups.push(payload);
      state.groups = [...new Set(state.groups)];
    },
    INSERT_CLUSTERNAME(state, payload) {
      state.clusterName = payload;
    },
    INSERT_CLUSTERPREFIX(state, payload) {
      state.clusterPrefix = payload;
    },
    INSERT_HOSTSPOWERED(state, payload) {
      state.hostsPowered = payload;
      state.hostsPowered = [...new Set(state.hostsPowered)];
    },
    INSERT_HOSTSDOWN(state, payload) {
      state.hostsDown = payload;
      state.hostsDown = [...new Set(state.hostsDown)];
    },
    INSERT_HOSTSUNKNOWN(state, payload) {
      state.hostsUnknown = payload;
      state.hostsUnknown = [...new Set(state.hostsUnknown)];
    },
    ADD_HOSTSPOWERED(state, payload) {
      state.hostsPowered.push(payload);
      state.hostsPowered = [...new Set(state.hostsPowered)];
    },
    ADD_HOSTSDOWN(state, payload) {
      state.hostsDown.push(payload);
      state.hostsDown = [...new Set(state.hostsDown)];
    },
    INSERT_HOSTSRESERVED(state, payload) {
      state.hostsReserved.splice(0);
      state.hostsReserved = payload;
      state.hostsReserved = [...new Set(state.hostsReserved)];
    },
    INSERT_HOSTSGRPRESERVED(state, payload) {
      state.hostsGrpReserved.splice(0);
      state.hostsGrpReserved = payload;
      state.hostsGrpReserved = [...new Set(state.hostsGrpReserved)];
    },
    INSERT_HOSTSOTHERRESERVED(state, payload) {
      state.hostsOtherReserved = payload;
      state.hostsOtherReserved = [...new Set(state.hostsOtherReserved)];
    },
    INSERT_HOSTSRESVPOW(state, payload) {
      state.hostsResvPow = payload;
      state.hostsResvPow = [...new Set(state.hostsResvPow)];
    },
    INSERT_HOSTSRESVDOWN(state, payload) {
      state.hostsResvDown = payload;
      state.hostsResvDown = [...new Set(state.hostsResvDown)];
    },
    INSERT_HOSTSRESVUNKNOWN(state, payload) {
      state.hostsResvUnknown = payload;
      state.hostsResvUnknown = [...new Set(state.hostsResvUnknown)];
    },
    INSERT_HOSTSGRPRESVPOW(state, payload) {
      state.hostsGrpResvPow = payload;
      state.hostsGrpResvPow = [...new Set(state.hostsGrpResvPow)];
    },
    INSERT_HOSTSGRPRESVDOWN(state, payload) {
      state.hostsGrpResvDown = payload;
      state.hostsGrpResvDown = [...new Set(state.hostsGrpResvDown)];
    },
    INSERT_HOSTSGRPRESVUNKNOWN(state, payload) {
      state.hostsGrpResvUnknown = payload;
      state.hostsGrpResvUnknown = [...new Set(state.hostsGrpResvUnknown)];
    },
    INSERT_HOSTSOTHERRESVPOW(state, payload) {
      state.hostsOtherResvPow = payload;
      state.hostsOtherResvPow = [...new Set(state.hostsOtherResvPow)];
    },
    INSERT_HOSTSOTHERRESVDOWN(state, payload) {
      state.hostsOtherResvDown = payload;
      state.hostsOtherResvDown = [...new Set(state.hostsOtherResvDown)];
    },
    INSERT_HOSTSOTHERRESVUNKOWN(state, payload) {
      state.hostsOtherResvUnknown = payload;
      state.hostsOtherResvUnknown = [...new Set(state.hostsOtherResvUnknown)];
    },
    INSERT_HOSTSAVLDOWN(state, payload) {
      state.hostsAvlDown = payload;
      state.hostsAvlDown = [...new Set(state.hostsAvlDown)];
    },
    INSERT_HOSTSAVLUNKNOWN(state, payload) {
      state.hostsAvlUnknown = payload;
      state.hostsAvlUnknown = [...new Set(state.hostsAvlUnknown)];
    },
    INSERT_HOSTSAVLPOW(state, payload) {
      state.hostsAvlPow = payload;
      state.hostsAvlPow = [...new Set(state.hostsAvlPow)];
    },
    INSERT_HOSTSBLOCKEDDOWN(state, payload) {
      state.hostsBlockedDown = payload;
      state.hostsBlockedDown = [...new Set(state.hostsBlockedDown)];
    },
    INSERT_HOSTSBLOCKEDUNKNOWN(state, payload) {
      state.hostsBlockedUnknown = payload;
      state.hostsBlockedUnknown = [...new Set(state.hostsBlockedUnknown)];
    },
    INSERT_HOSTSBLOCKEDPOW(state, payload) {
      state.hostsBlockedPow = payload;
      state.hostsBlockedPow = [...new Set(state.hostsBlockedPow)];
    },
    INSERT_HOSTSBLOCKED(state, payload) {
      state.hostsBlocked = payload;
      state.hostsBlocked = [...new Set(state.hostsBlocked)];
    },
    INSERT_HOSTSRESTRICTEDPOW(state, payload) {
      state.hostsRestrictedPow = payload;
      state.hostsRestrictedPow = [...new Set(state.hostsRestrictedPow)];
    },
    INSERT_HOSTSRESTRICTEDDOWN(state, payload) {
      state.hostsRestrictedDown = payload;
      state.hostsRestrictedDown = [...new Set(state.hostsRestrictedDown)];
    },
    INSERT_HOSTSRESTRICTEDUNKNOWN(state, payload) {
      state.hostsRestrictedUnknown = payload;
      state.hostsRestrictedUnknown = [...new Set(state.hostsRestrictedUnknown)];
    },
    INSERT_HOSTSINSTERRDOWN(state, payload) {
      state.hostsInstErrDown = payload;
      state.hostsInstErrDown = [...new Set(state.hostsInstErrDown)];
    },
    INSERT_HOSTSINSTERRUNKNOWN(state, payload) {
      state.hostsInstErrUnknown = payload;
      state.hostsInstErrUnknown = [...new Set(state.hostsInstErrUnknown)];
    },

    INSERT_HOSTSINSTERRPOW(state, payload) {
      state.hostsInstErrPow = payload;
      state.hostsInstErrPow = [...new Set(state.hostsInstErrPow)];
    },
    INSERT_HOSTSINSTERR(state, payload) {
      state.hostsInstErr = payload;
      state.hostsInstErr = [...new Set(state.hostsInstErr)];
    },
    INSERT_HOSTSFORRESV(state, payload) {
      state.hostsForResv = payload;
      state.hostsForResv = [...new Set(state.hostsForResv)];
    },
    INSERT_GROUPS(state, payload) {
      state.groups = payload;
      state.groups = [...new Set(state.groups)];
    },
    INSERT_GROUPNAMES(state, payload) {
      state.groupNames = payload;
      state.groupNames = [...new Set(state.groupNames)];
    },
    INSERT_OWNER_GROUP_NAMES(state, payload) {
      state.ownerGroupNames = payload;
      state.ownerGroupNames = [...new Set(state.ownerGroupNames)];
    },
    INSERT_MEMBER_GROUP_NAMES(state, payload) {
      state.memberGroupNames = payload;
      state.memberGroupNames = [...new Set(state.memberGroupNames)];
    },
    INSERT_NEWOWNER_GROUP_NAME(state, payload) {
      state.ownerGroupNames.push(payload);
      state.ownerGroupNames = [...new Set(state.ownerGroupNames)];
    },
    INSERT_PROFILES(state, payload) {
      state.profiles = payload;
      state.profiles = [...new Set(state.profiles)];
    },
    INSERT_DISTROS(state, payload) {
      state.distros = payload;
      state.distros = [...new Set(state.distros)];
    },
    INSERT_EDISTRONAMES(state, payload) {
      state.eDistroNames = payload;
      state.eDistroNames = [...new Set(state.eDistroNames)];
    },
    INSERT_EPROFILENAMES(state, payload) {
      state.eProfileNames = payload;
      state.eProfileNames = [...new Set(state.eProfileNames)];
    },
    INSERT_USERRESERVATIONS(state, payload) {
      state.userReservations = payload;
      state.userReservations = [...new Set(state.userReservations)];
    },
    INSERT_GROUPRESERVATIONS(state, payload) {
      state.groupReservations = payload;
      state.groupReservations = [...new Set(state.groupReservations)];
    },
    INSERT_ASSOCIATEDRESERVATIONS(state, payload) {
      state.associatedReservations = payload;
      state.associatedReservations = [...new Set(state.associatedReservations)];
    },
    INSERT_NEWUSERRESERVATIONS(state, payload) {
      state.userReservations.push(payload);
      state.userReservations = [...new Set(state.userReservations)];
    },
    INSERT_USERS(state, payload) {
      state.users = payload;
      state.users = [...new Set(state.users)];
    },
    INSERT_USERDETAILS(state, payload) {
      state.userDetails = payload;
      state.userDetails = [...new Set(state.userDetails)];
    },
    INSERT_MOTD(state, payload) {
      state.motd = payload;
    },
    INSERT_MOTDFLAG(state, payload) {
      state.motdFlag = payload;
    },

    // Remove/Delete Hosts, Reservations, Profile, Distro
    REMOVE_HOSTSRESVPOW(state, payload) {
      state.hostsAvlPow.push(payload.value);
      state.hostsResvPow.splice(payload.key, 1);
    },
    REMOVE_HOSTSRESVDOWN(state, payload) {
      state.hostsAvlDown.push(payload.value);
      state.hostsResvDown.splice(payload.key, 1);
    },
    REMOVE_HOSTSRESVUNKNOWN(state, payload) {
      state.hostsAvlUnknown.push(payload.value);
      state.hostsResvUnknown.splice(payload.key, 1);
    },
    REMOVE_HOSTSDOWN(state, payload) {
      const index = state.hostsDown.findIndex((host) => host === payload);
      if (~index) {
        state.hostsDown.splice(index, 1);
      }
    },
    REMOVE_HOSTSPOWERED(state, payload) {
      const index = state.hostsPowered.findIndex((host) => host === payload);
      if (~index) {
        state.hostsPowered.splice(index, 1);
      }
    },
    REMOVE_HOSTSUNKNOWN(state, payload) {
      const index = state.hostsUnknown.findIndex((host) => host === payload);
      if (~index) {
        state.hostsUnknown.splice(index, 1);
      }
    },
    REMOVE_HOSTSINSTERRPOW(state, payload) {
      state.hostsAvlPow.push(payload.value);
      state.hostsInstErrPow.splice(payload.key, 1);
    },
    REMOVE_HOSTSINSTERRDOWN(state, payload) {
      state.hostsAvlDown.push(payload.value);
      state.hostsInstErrDown.splice(payload.key, 1);
    },
    REMOVE_HOSTSINSTERRUNKNOWN(state, payload) {
      state.hostsAvlUnknown.push(payload.value);
      state.hostsInstErrUnknown.splice(payload.key, 1);
    },
    REMOVE_HOSTSAVLPOW(state, payload) {
      state.hostsResvPow.push(payload.value);
      state.hostsAvlPow.splice(payload.key, 1);
    },
    REMOVE_HOSTSAVLDOWN(state, payload) {
      state.hostsResvDown.push(payload.value);
      state.hostsAvlDown.splice(payload.key, 1);
    },
    REMOVE_HOSTSAVLUNKNOWN(state, payload) {
      state.hostsResvUnknown.push(payload.value);
      state.hostsAvlUnknown.splice(payload.key, 1);
    },
    ADD_HOSTSFORRESV(state, payload) {
      state.hostsForResv.push(payload);
    },
    REMOVE_HOSTSFORRESV(state, payload) {
      state.hostsForResv.splice(payload, 1);
    },
    UPDATE_HOSTSINSTERRPOW(state, payload) {
      state.hostsInstErrPow.push(payload.value);
      state.hostsAvlPow.splice(payload.key, 1);
    },
    UPDATE_HOSTSINSTERRDOWN(state, payload) {
      state.hostsInstErrDown.push(payload.value);
      state.hostsAvlDown.splice(payload.key, 1);
    },
    UPDATE_HOSTSINSTERRUNKNOWN(state, payload) {
      state.hostsInstErrUnknown.push(payload.value);
      state.hostsAvlUnknown.splice(payload.key, 1);
    },

    DELETE_USERRESERVATION(state, payload) {
      state.userReservations.splice(payload, 1);
    },
    DELETE_ASSOCIATEDRESERVATION(state, payload) {
      state.associatedReservations.splice(payload, 1);
    },
    DELETE_RESERVATION(state, payload) {
      state.reservations.splice(payload, 1);
    },
    DELETE_DISTRO(state, payload) {
      state.distros.splice(payload, 1);
    },
    DELETE_EDISTRO_NAMES(state, payload) {
      state.eDistroNames.splice(payload, 1);
    },
    DELETE_GROUP(state, payload) {
      state.groups.splice(payload, 1);
    },
    DELETE_OWNER_GROUP_NAME(state, payload) {
      state.ownerGroupNames.splice(payload, 1);
    },
    DELETE_PROFILE(state, payload) {
      state.profiles.splice(payload, 1);
    },
    EXTEND_MAX(state, payload) {
      state.userReservations[payload.key].remainHours =
        payload.value.remainHours;
      state.userReservations[payload.key].end = payload.value.end;
    },
    EXTEND_MAXASC(state, payload) {
      state.associatedReservations[payload.key].remainHours =
        payload.value.remainHours;
      state.associatedReservations[payload.key].end = payload.value.end;
    },
    EXTEND_MAXALL(state, payload) {
      state.reservations[payload.key].remainHours = payload.value.remainHours;
      state.userReservations[payload.key].end = payload.value.end;
    },
    SAVE_RESV(state, payload) {
      state.userReservations[payload.key].name = payload.value.name;
      state.userReservations[payload.key].description =
        payload.value.description;
      state.userReservations[payload.key].owner = payload.value.owner;
      state.userReservations[payload.key].group = payload.value.group;
    },
    SAVE_ASC_RESV(state, payload) {
      state.associatedReservations[payload.key].name = payload.value.name;
      state.associatedReservations[payload.key].description =
        payload.value.description;
      state.associatedReservations[payload.key].owner = payload.value.owner;
      state.associatedReservations[payload.key].group = payload.value.group;
    },
    SAVE_RESVALL(state, payload) {
      state.reservations[payload.key].name = payload.value.name;
      state.reservations[payload.key].description = payload.value.description;
      state.reservations[payload.key].owner = payload.value.owner;
      state.reservations[payload.key].group = payload.value.group;
    },
    SELECTED_RESVHOSTS(state, payload) {
      state.selectedHosts = [];
      state.selectedHosts = payload.split(",");
      state.selectedHosts = [...new Set(state.selectedHosts)];
      
    },
    SELECTED_RESVHOSTS_COUNT(state, payload) {
      // state.selectedHostsCount = 0;
      state.selectedHostsCount = payload;      
    },
    SELECTED_RESVHOSTID(state, payload) {
      state.selectedHosts = [];
      state.selectedHostID = payload;
      state.selectedHostID = [...new Set(state.selectedHostID)];
    },
    SELECTED_POW(state, payload) {
      state.hostSelectedPow = [];
      state.hostSelectedPow = payload;
    },
    SELECTED_DOWN(state, payload) {
      state.hostSelectedDown = [];
      state.hostSelectedDown = payload;
    },
    SELECTED_UNKNOWN(state, payload) {
      state.hostSelectedUnknown = payload;
    },
    SAVE_DISTRO(state, payload) {
      state.userReservations[payload.key].distro = payload.value.distro;
    },
    SAVE_DISTROASC(state, payload) {
      state.associatedReservations[payload.key].distro = payload.value.distro;
    },
    SAVE_DISTROALL(state, payload) {
      state.reservations[payload.key].distro = payload.value.distro;
    },
    SAVE_RESVNEWPROFILE(state, payload) {
      state.userReservations[payload.key].profile = payload.value.profile;
    },
    SAVE_PROFILEASC(state, payload) {
      state.associatedReservations[payload.key].profile = payload.value.profile;
    },
    SAVE_PROFILEALL(state, payload) {
      state.reservations[payload.key].profile = payload.value.profile;
    },
    SAVE_GROUP(state, payload) {
      state.groups[payload.key].name = payload.value.name;
      state.groups[payload.key].description = payload.value.description;
      state.groups[payload.key].owner = payload.value.owner;
      state.groups[payload.key].members = payload.value.members;
    },
    SAVE_UPDATED_DISTRO(state, payload) {
      state.distros[payload.key].name = payload.value.name;
      state.distros[payload.key].description = payload.value.description;
      state.distros[payload.key].kernelArgs = payload.value.kernelArgs;
      state.distros[payload.key].groups = payload.value.groups;
    },
    SAVE_PROFILE(state, payload) {
      state.profiles[payload.key].name = payload.value.name;
      state.profiles[payload.key].description = payload.value.description;
      state.profiles[payload.key].kernelArgs = payload.value.kernelArgs;
    },
    SAVE_HOSTS(state, payload) {
      state.userReservations[payload.key].hosts = payload.value.hosts;
    },
    SAVE_HOSTSALL(state, payload) {
      state.reservations[payload.key].hosts = payload.value.hosts;
    },
    SAVE_HOSTRANGE(state, payload) {
      state.userReservations[payload.key].hostRange = payload.value.hostRange;
    },
    SAVE_HOSTRANGEASC(state, payload) {
      state.associatedReservations[payload.key].hostRange =
        payload.value.hostRange;
    },
    SAVE_HOSTRANGEALL(state, payload) {
      state.reservations[payload.key].hostRange = payload.value.hostRange;
    },
    ADD_EDISTRONAMES(state, payload) {
      state.eDistroNames.push(payload);
    },
    INSERT_ERROR(state, error) {
      state.errors.push(error);
    },
    DEFAULTRESERVATIONMINUTES(state, payload) {
      state.defaultReserveMinutes = payload;
    },
    VLANMIN(state, payload) {
      state.vlanMin = payload;
    },
    VLANMAX(state, payload) {
      state.vlanMax = payload;
    },
  },
  actions: {
    // Authentication
    login({ commit }, user) {
      const uname = user.username;
      const session_url = Vue.prototype.$config.IGOR_API_BASE_URL + "/login";
      return new Promise((resolve, reject) => {
        commit("auth_request");
        axios
          .post(session_url, null, {
            //AxiosRequestConfig parameter
            withCredentials: true,
            auth: {
              username: user.username,
              password: user.password,
            },
          })
          .then(function(resp) {
            commit("auth_success", uname);
            sessionStorage.setItem("username", uname);
            sessionStorage.setItem("authenticated", true);
            resolve(resp);
          })
          .catch(function(err) {
            commit("auth_error");
            reject(err);
            console.log("Error on Authentication");
          });
      });
    },
    register({ commit }, user) {
      return new Promise((resolve, reject) => {
        commit("auth_request");
        axios({
          url: Vue.prototype.$config.IGOR_API_BASE_URL + "/login",
          data: user,
          method: "GET",
        })
          .then((resp) => {
            const token = resp.data.token;
            const user = resp.data.user;
            localStorage.setItem("token", token);
            axios.defaults.headers.common["Authorization"] = token;
            commit("auth_success", token, user);
            resolve(resp);
          })
          .catch((err) => {
            commit("auth_error", err);
            localStorage.removeItem("token");
            reject(err);
          });
      });
    },
    logout({ commit }) {
      return new Promise((resolve, reject) => {
        commit("logout");
        resolve();
      });
    },
    loggedIn({ commit }, payload) {
      commit("LOGGED_IN", payload);
    },
    saveServerTime({ commit }, payload) {
      try {
        commit("SAVE_SERVER_TIME", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },

    // Add New Reservations, Profile, Distro, Hosts
    insertHosts({ commit }, payload) {
      try {
        commit("INSERT_HOSTS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostNames({ commit }, payload) {
      try {
        commit("INSERT_HOSTNAMES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertReservations({ commit }, payload) {
      try {
        commit("INSERT_RESERVATIONS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertReservationsForFiltering({ commit }, payload) {
      try {
        commit("INSERT_RESERVATIONS_FOR_FILTERING", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertNewReservations({ commit }, payload) {
      try {
        commit("INSERT_NEWRESERVATIONS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertNewProfile({ commit }, payload) {
      try {
        commit("INSERT_NEWPROFILE", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertNewDistro({ commit }, payload) {
      try {
        commit("INSERT_NEWDISTRO", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertActiveProfiles({ commit }, payload) {
      try {
        commit("INSERT_ACTIVE_PROFILES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertActiveDistros({ commit }, payload) {
      try {
        commit("INSERT_ACTIVE_DISTROS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertNewGroup({ commit }, payload) {
      try {
        commit("INSERT_NEWGROUP", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertNewGroupName({ commit }, payload) {
      try {
        commit("INSERT_NEWOWNER_GROUP_NAME", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertClusterName({ commit }, payload) {
      try {
        commit("INSERT_CLUSTERNAME", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertClusterPrefix({ commit }, payload) {
      try {
        commit("INSERT_CLUSTERPREFIX", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsPowered({ commit }, payload) {
      try {
        commit("INSERT_HOSTSPOWERED", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    removeHostsUnknown({ commit }, payload) {
      try {
        commit("REMOVE_HOSTSUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    removeHostsPowered({ commit }, payload) {
      try {
        commit("REMOVE_HOSTSPOWERED", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    removeHostsDown({ commit }, payload) {
      try {
        commit("REMOVE_HOSTSDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    addHostsPowered({ commit }, payload) {
      try {
        commit("ADD_HOSTSPOWERED", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    addHostsDown({ commit }, payload) {
      try {
        commit("ADD_HOSTSDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsReserved({ commit }, payload) {
      try {
        commit("INSERT_HOSTSRESERVED", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsGrpReserved({ commit }, payload) {
      try {
        commit("INSERT_HOSTSGRPRESERVED", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsOtherReserved({ commit }, payload) {
      try {
        commit("INSERT_HOSTSOTHERRESERVED", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsResvPow({ commit }, payload) {
      try {
        commit("INSERT_HOSTSRESVPOW", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsResvDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSRESVDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsResvUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSRESVUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsGrpResvPow({ commit }, payload) {
      try {
        commit("INSERT_HOSTSGRPRESVPOW", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsGrpResvDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSGRPRESVDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsGrpResvUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSGRPRESVUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsOtherResvPow({ commit }, payload) {
      try {
        commit("INSERT_HOSTSOTHERRESVPOW", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsOtherResvDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSOTHERRESVDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsOtherResvUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSOTHERRESVUNKOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsAvlDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSAVLDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsAvlUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSAVLUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsAvlPow({ commit }, payload) {
      try {
        commit("INSERT_HOSTSAVLPOW", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsRestrictedPow({ commit }, payload) {
      try {
        commit("INSERT_HOSTSRESTRICTEDPOW", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsRestrictedDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSRESTRICTEDDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsRestrictedUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSRESTRICTEDUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsBlockedDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSBLOCKEDDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsBlockedUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSBLOCKEDUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsBlockedPow({ commit }, payload) {
      try {
        commit("INSERT_HOSTSBLOCKEDPOW", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsBlocked({ commit }, payload) {
      try {
        commit("INSERT_HOSTSBLOCKED", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsInstErrDown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSINSTERRDOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsInstErrUnknown({ commit }, payload) {
      try {
        commit("INSERT_HOSTSINSTERRUNKNOWN", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsInstErrPow({ commit }, payload) {
      try {
        commit("INSERT_HOSTSINSTERRPOW", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsInstErr({ commit }, payload) {
      try {
        commit("INSERT_HOSTSINSTERR", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertHostsForResv({ commit }, payload) {
      try {
        commit("INSERT_HOSTSFORRESV", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertGroups({ commit }, payload) {
      try {
        commit("INSERT_GROUPS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertGroupNames({ commit }, payload) {
      try {
        commit("INSERT_GROUPNAMES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertOwnerGroupNames({ commit }, payload) {
      try {
        commit("INSERT_OWNER_GROUP_NAMES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertMemberGroupNames({ commit }, payload) {
      try {
        commit("INSERT_MEMBER_GROUP_NAMES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertProfiles({ commit }, payload) {
      try {
        commit("INSERT_PROFILES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertDistros({ commit }, payload) {
      try {
        commit("INSERT_DISTROS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertEDistroNames({ commit }, payload) {
      try {
        commit("INSERT_EDISTRONAMES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertEProfileNames({ commit }, payload) {
      try {
        commit("INSERT_EPROFILENAMES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertUserReservations({ commit }, payload) {
      try {
        commit("INSERT_USERRESERVATIONS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertGroupReservations({ commit }, payload) {
      try {
        commit("INSERT_GROUPRESERVATIONS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertAssociatedReservations({ commit }, payload) {
      try {
        commit("INSERT_ASSOCIATEDRESERVATIONS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertNewUserReservations({ commit }, payload) {
      try {
        commit("INSERT_NEWUSERRESERVATIONS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertUsers({ commit }, payload) {
      try {
        commit("INSERT_USERS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertUserDetails({ commit }, payload) {
      try {
        commit("INSERT_USERDETAILS", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertMotd({ commit }, payload) {
      try {
        commit("INSERT_MOTD", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },
    insertMotdFlag({ commit }, payload) {
      try {
        commit("INSERT_MOTDFLAG", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },

    // Delete user reservation
    removeHostsResvPow({ commit }, payload) {
      commit("REMOVE_HOSTSRESVPOW", payload);
    },
    removeHostsResvDown({ commit }, payload) {
      commit("REMOVE_HOSTSRESVDOWN", payload);
    },
    removeHostsResvUnknown({ commit }, payload) {
      commit("REMOVE_HOSTSRESVUNKNOWN", payload);
    },
    removeHostsInstErrPow({ commit }, payload) {
      commit("REMOVE_HOSTSINSTERRPOW", payload);
    },
    removeHostsInstErrDown({ commit }, payload) {
      commit("REMOVE_HOSTSINSTERRDOWN", payload);
    },
    removeHostsInstErrUnknown({ commit }, payload) {
      commit("REMOVE_HOSTSINSTERRUNKNOWN", payload);
    },
    addHostsForResv({ commit }, payload) {
      commit("ADD_HOSTSFORRESV", payload);
    },
    deleteUserReservation({ commit }, payload) {
      commit("DELETE_USERRESERVATION", payload);
    },
    deleteAssociatedReservation({ commit }, payload) {
      commit("DELETE_ASSOCIATEDRESERVATION", payload);
    },
    deleteReservation({ commit }, payload) {
      commit("DELETE_RESERVATION", payload);
    },
    deleteDistro({ commit }, payload) {
      commit("DELETE_DISTRO", payload);
    },
    deleteEDistroNames({ commit }, payload) {
      commit("DELETE_EDISTRO_NAMES", payload);
    },
    deleteGroup({ commit }, payload) {
      commit("DELETE_GROUP", payload);
    },
    deleteOwnerGroupName({ commit }, payload) {
      commit("DELETE_OWNER_GROUP_NAME", payload);
    },
    deleteProfile({ commit }, payload) {
      commit("DELETE_PROFILE", payload);
    },

    // Extend Reservations - Max
    extendMax({ commit }, payload) {
      commit("EXTEND_MAX", payload);
    },
    extendMaxAsc({ commit }, payload) {
      commit("EXTEND_MAXASC", payload);
    },
    extendMaxAll({ commit }, payload) {
      commit("EXTEND_MAXALL", payload);
    },

    // Edit Reservations
    saveResv({ commit }, payload) {
      commit("SAVE_RESV", payload);
    },
    saveAscResv({ commit }, payload) {
      commit("SAVE_ASC_RESV", payload);
    },
    saveResvAll({ commit }, payload) {
      commit("SAVE_RESVALL", payload);
    },
    selectedResvHosts({ commit }, payload) {
      commit("SELECTED_RESVHOSTS", payload);
    },
    selectedResvHostsCount({ commit }, payload) {
      commit("SELECTED_RESVHOSTS_COUNT", payload);
    },
    selectedResvHostID({ commit }, payload) {
      commit("SELECTED_RESVHOSTID", payload);
    },
    selectedPow({ commit }, payload) {
      commit("SELECTED_POW", payload);
    },
    selectedDown({ commit }, payload) {
      commit("SELECTED_DOWN", payload);
    },
    selectedUnknown({ commit }, payload) {
      commit("SELECTED_UNKNOWN", payload);
    },
    saveDistro({ commit }, payload) {
      commit("SAVE_DISTRO", payload);
    },
    saveDistroAsc({ commit }, payload) {
      commit("SAVE_DISTROASC", payload);
    },
    saveDistroAll({ commit }, payload) {
      commit("SAVE_DISTROALL", payload);
    },
    saveResvNewProfile({ commit }, payload) {
      commit("SAVE_RESVNEWPROFILE", payload);
    },
    saveProfileAsc({ commit }, payload) {
      commit("SAVE_PROFILEASC", payload);
    },
    saveProfileAll({ commit }, payload) {
      commit("SAVE_PROFILEALL", payload);
    },
    saveGroup({ commit }, payload) {
      commit("SAVE_GROUP", payload);
    },
    saveUpdatedDistro({ commit }, payload) {
      commit("SAVE_UPDATED_DISTRO", payload);
    },
    saveProfile({ commit }, payload) {
      commit("SAVE_PROFILE", payload);
    },
    saveHosts({ commit }, payload) {
      commit("SAVE_HOSTS", payload);
    },
    saveHostsAll({ commit }, payload) {
      commit("SAVE_HOSTSALL", payload);
    },
    saveHostRange({ commit }, payload) {
      commit("SAVE_HOSTRANGE", payload);
    },
    saveHostRangeAsc({ commit }, payload) {
      commit("SAVE_HOSTRANGEASC", payload);
    },
    saveHostRangeAll({ commit }, payload) {
      commit("SAVE_HOSTRANGEALL", payload);
    },

    addEDistroNames({ commit }, payload) {
      try {
        commit("ADD_EDISTRONAMES", payload);
      } catch (error) {
        commit("INSERT_ERROR", error);
      }
    },

    // Add New reservations
    removeHostsAvlPow({ commit }, payload) {
      commit("REMOVE_HOSTSAVLPOW", payload);
    },
    removeHostsAvlDown({ commit }, payload) {
      commit("REMOVE_HOSTSAVLDOWN", payload);
    },
    removeHostsAvlUnknown({ commit }, payload) {
      commit("REMOVE_HOSTSAVLUNKNOWN", payload);
    },
    updateHostsInstErrPow({ commit }, payload) {
      commit("UPDATE_HOSTSINSTERRPOW", payload);
    },
    updateHostsInstErrDown({ commit }, payload) {
      commit("UPDATE_HOSTSINSTERRDOWN", payload);
    },
    updateHostsInstErrUnknown({ commit }, payload) {
      commit("UPDATE_HOSTSINSTERRUNKNOWN", payload);
    },
    removeHostsForResv({ commit }, payload) {
      commit("REMOVE_HOSTSFORRESV", payload);
    },

    // Config parameters
    defaultReserveMinutes({ commit }, payload) {
      commit("DEFAULTRESERVATIONMINUTES", payload);
    },

    vlanMin({ commit }, payload) {
      commit("VLANMIN", payload);
    },

    vlanMax({ commit }, payload) {
      commit("VLANMAX", payload);
    },
  },
  getters: {
    // Authentication
    username(state) {
      return state.username;
    },
    isLoggedIn(state) {
      return state.isLoggedIn;
    },
    authStatus: (state) => state.status,

    serverTime(state) {
      return state.serverTime;
    },

    // Reservation, Host, Profile, Distro
    hostNames(state) {
      return state.hostNames;
    },
    hosts(state) {
      return state.hosts;
    },
    reservations(state) {
      return state.reservations;
    },
    reservationsFilteredLength(state) {
      return state.reservationsFilteredLength;
    },
    clusterName(state) {
      return state.clusterName;
    },
    clusterPrefix(state) {
      return state.clusterPrefix;
    },
    hostsPowered(state) {
      return state.hostsPowered;
    },
    hostsDown(state) {
      return state.hostsDown;
    },
    hostsUnknown(state) {
      return state.hostsUnknown;
    },
    hostsReserved(state) {
      return state.hostsReserved;
    },
    hostsOtherReserved(state) {
      return state.hostsOtherReserved;
    },
    hostsResvPow(state) {
      return state.hostsResvPow;
    },
    hostsResvDown(state) {
      return state.hostsResvDown;
    },
    hostsResvUnknown(state) {
      return state.hostsResvUnknown;
    },
    hostsOtherResvPow(state) {
      return state.hostsOtherResvPow;
    },
    hostsOtherResvDown(state) {
      return state.hostsOtherResvDown;
    },
    hostsOtherResvUnknown(state) {
      return state.hostsOtherResvUnknown;
    },
    hostsGrpResvPow(state) {
      return state.hostsGrpResvPow;
    },
    hostsGrpResvDown(state) {
      return state.hostsGrpResvDown;
    },
    hostsGrpResvUnknown(state) {
      return state.hostsGrpResvUnknown;
    },
    hostsAvlPow(state) {
      return state.hostsAvlPow;
    },
    hostsAvlDown(state) {
      return state.hostsAvlDown;
    },
    hostsAvlUnknown(state) {
      return state.hostsAvlUnknown;
    },
    hostsRestrictedPow(state) {
      return state.hostsRestrictedPow;
    },
    hostsRestrictedDown(state) {
      return state.hostsRestrictedDown;
    },
    hostsRestrictedUnknown(state) {
      return state.hostsRestrictedUnknown;
    },
    hostsBlockedPow(state) {
      return state.hostsBlockedPow;
    },
    hostsBlockedDown(state) {
      return state.hostsBlockedDown;
    },
    hostsBlockedUnknown(state) {
      return state.hostsBlockedUnknown;
    },
    hostsBlocked(state) {
      return state.hostsBlocked;
    },
    hostsInstErrPow(state) {
      return state.hostsInstErrPow;
    },
    hostsInstErrDown(state) {
      return state.hostsInstErrDown;
    },
    hostsInstErrUnknown(state) {
      return state.hostsInstErrUnknown;
    },
    hostsInstErr(state) {
      return state.hostsInstErr;
    },
    hostsForResv(state) {
      return state.hostsForResv;
    },
    selectedHosts(state){
      return state.selectedHosts;
    },
    selectedHostsCount(state){
      return state.selectedHostsCount;
    },
    selectedHostID(state){
      return state.selectedHostID;
    },
    hostSelectedPow(state){
      return state.hostSelectedPow;
    },
    hostSelectedDown(state){
      return state.hostSelectedDown;
    },
    hostSelectedUnknown(state){
      return state.hostSelectedUnknown;
    },
    profiles(state) {
      return state.profiles;
    },
    distros(state) {
      return state.distros;
    },
    activeProfiles(state) {
      return state.activeProfiles;
    },
    activeDistros(state) {
      return state.activeDistros;
    },
    eDistroNames(state) {
      return state.eDistroNames;
    },
    eProfileNames(state) {
      return state.eProfileNames;
    },
    groups(state) {
      return state.groups;
    },
    groupNames(state) {
      return state.groupNames;
    },
    ownerGroupNames(state) {
      return state.ownerGroupNames;
    },
    memberGroupNames(state) {
      return state.ownerGroupNames;
    },
    userReservations(state) {
      return state.userReservations;
    },
    groupReservations(state) {
      return state.groupReservations;
    },
    associatedReservations(state) {
      return state.associatedReservations;
    },
    users(state) {
      return state.users;
    },
    userDetails(state) {
      return state.userDetails;
    },
    motd(state) {
      return state.motd;
    },
    motdFlag(state) {
      return state.motdFlag;
    },
    defaultReserveMinutes(state) {
      return state.defaultReserveMinutes;
    },
    vlanMin(state) {
      return state.vlanMin;
    },
    vlanMax(state) {
      return state.vlanMax;
    },
  },
});
