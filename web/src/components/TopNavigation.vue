<template>
  <div>
    <!-- Modal for editing profile -->
    <div>
      <b-modal ref="editModal" hide-footer title="Edit Profile">
        <div class="container">
          <b-card no-body>
            <b-tabs
              card
              active-nav-item-class="font-weight-bold text-dark"
              active-tab-class="text-dark"
              title-item-class="text-dark"
            >
              <!-- Profile Tab -->
              <b-tab
                active
                no-body
                title="Details"
                title-link-class="text-dark"
              >
                <div class="container">
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="name-group"
                        class="form-group col-6"
                        label-for="name"
                        label="Name"
                      >
                        <b-form-input
                          id="name"
                          placeholder="Name"
                          v-model="editUser.name"
                          class="form-control"
                        ></b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="email-group"
                        class="form-group col-6"
                        label-for="email"
                        label="Email"
                      >
                        <b-form-input
                          id="email"
                          placeholder="Email"
                          type="email"
                          v-model="editUser.email"
                          class="form-control"
                        >
                        </b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="saveProfile"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="clearEditData"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </b-tab>
              <!-- Password Reset Tab -->
              <b-tab
                no-body
                title="Password"
                title-link-class="text-dark"
                v-if="!ldapEnabled"
              >
                <div class="container">
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="oldPassword-group"
                        class="form-group col-6"
                        label-for="oldPassword"
                        label="Password"
                      >
                        <b-form-input
                          id="oldPassword"
                          type="password"
                          placeholder="Password"
                          v-model="editPassword.oldPassword"
                          class="form-control"
                        ></b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="password-group"
                        class="form-group col-6"
                        label-for="password"
                        label="New Password"
                      >
                        <b-form-input
                          id="password"
                          placeholder="New Password"
                          type="password"
                          v-model="editPassword.password"
                          class="form-control"
                        >
                        </b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="savePassword"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="clearEditData"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </b-tab>
            </b-tabs>
          </b-card>
        </div>
      </b-modal>
    </div>
    <!-- Top Panel -->
    <b-container fluid="true">
      <b-row>
        <b-col class="ml-2 col-sm-auto">
          <h2 class="font-weight-bold title-brand">Igor 2.2</h2>
        </b-col>
        <b-col align-v="bottom">
          <span v-if="isLoggedIn">
            <div>
              <h5
                class="font-weight-bold text-center"
                id="motdMsg"
                v-bind:class="{
                  'text-primary': true,
                  'text-danger': motdFlag,
                }"
              >
                {{ motd }}
              </h5>
            </div>
          </span>
        </b-col>
        <b-col class="mr-3 col-sm-auto" align-v="bottom">
          <b-navbar v-if="isLoggedIn">
            <b-navbar-nav
              class="btn bg-transparent btn-outline-light text-dark buttonfocus"
            >
              <b-nav-item-dropdown right>
                <!-- Using 'button-content' slot -->
                <template #button-content>
                  <span class="text-dark">
                    <b-icon-person-circle
                      aria-hidden="true"
                      variant="warning"
                      scale="1.5"
                      class="mr-2"
                    >
                    </b-icon-person-circle>
                    {{ username }}
                  </span>
                </template>
                <b-dropdown-item href="#" @click="updateProfile"
                  >Account</b-dropdown-item
                >
                <b-dropdown-item href="#" @click="logout"
                  >Sign Out</b-dropdown-item
                >
              </b-nav-item-dropdown>
            </b-navbar-nav>
          </b-navbar>
        </b-col>
      </b-row>
    </b-container>
  </div>
</template>

<script>
import axios from "axios";
export default {
  data() {
    return {
      editUser: {
        name: "",
        email: "",
      },
      editPassword: {
        oldPassword: "",
        password: "",
      },
      name: "",
      email: "",
      ldapEnabled: false,
    };
  },

  mounted() {
    let configUrl = this.$config.IGOR_API_BASE_URL + "/config/public";
    axios.get(configUrl).then((response) => {
      this.ldapEnabled = !response.data.data.igor.localAuthEnabled;
      this.$store.dispatch(
        "defaultReserveMinutes",
        response.data.data.igor.defaultReserveMinutes
      );
      this.$store.dispatch(
        "vlanMin",
        response.data.data.igor.vlanRangeMin
      );
      this.$store.dispatch(
        "vlanMax",
        response.data.data.igor.vlanRangeMax
      );
    });
  },
  computed: {
    isLoggedIn() {
      return this.$store.getters.isLoggedIn;
    },
    username() {
      return sessionStorage.getItem("username");
    },

    motd() {
      return this.$store.getters.motd;
    },
    motdFlag() {
      return this.$store.getters.motdFlag;
    },
  },
  methods: {
    logout() {
      this.$store.dispatch("logout").then(() => {
        this.$router.push("/login");
        window.location.reload();
      });
    },

    // Axios global interceptors to keep track of token
    // Also, check if refresh token is available : Need to confirm this
    created: function() {
      this.$http.interceptors.response.use(undefined, function(err) {
        return new Promise(function(resolve, reject) {
          if (
            err.response.status === 401 &&
            err.config &&
            !err.config.__isRetryRequest
          ) {
            this.sessionExpired();
          }
          throw err;
        });
      });
    },

    // Redirect to Login on Session expiration
    sessionExpired() {
      this.$store.dispatch("logout").then(() => {
        this.$router.push("/login");
        window.location.reload();
      });
    },

    updateProfile() {
      let userDetailsUrl =
        this.$config.IGOR_API_BASE_URL + "/users/" + "?name=" + this.username;
      axios.get(userDetailsUrl, { withCredentials: true }).then((response) => {
        this.editUser.email = response.data.data[0].email;
        this.email = response.data.data[0].email;
        this.name = response.data.data[0].fullName;
        this.editUser.name = response.data.data[0].fullName;
      });
      this.$refs.editModal.show();
    },
    saveProfile() {
      let saveProfileUrl =
        this.$config.IGOR_API_BASE_URL + "/users/" + this.username;
      let editData = {
        fullName: this.editUser.name,
        email: this.editUser.email,
      };
      // Sanity check
      if (this.editUser.name === this.name) {
        this.$delete(editData, "fullName");
      }
      if (this.editUser.email === this.email) {
        this.$delete(editData, "email");
      }

      axios
        .patch(saveProfileUrl, editData, { withCredentials: true })
        .then((response) => {
          alert("User profile updated successfully!");
          this.clearEditData();
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
    savePassword() {
      let saveProfileUrl =
        this.$config.IGOR_API_BASE_URL + "/users/" + this.username;

      axios
        .patch(saveProfileUrl, this.editPassword, { withCredentials: true })
        .then((response) => {
          alert("Password updated successfully! Logging Out!");
          this.clearEditData();
          this.$refs.editModal.hide();
          this.logout();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
    clearEditData() {
      this.editUser = {
        name: "",
        email: "",
      };
      this.editPassword = {
        oldPassword: "",
        password: "",
      };
      this.$refs.editModal.hide();
    },
  },
};
</script>
