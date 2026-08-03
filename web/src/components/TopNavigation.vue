<template>
  <div>
    <!-- Modal for editing profile -->
    <div>
      <b-modal ref="editModal" hide-footer title="Edit Profile">
        <div class="container">
          <b-card no-body>
            <b-tabs
              card
              active-nav-item-class="fw-bold text-dark"
              active-tab-class=""
              title-item-class=""
            >
              <!-- Profile Tab -->
              <b-tab
                active
                no-body
                title="Details"
                title-link-class=""
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
                title-link-class=""
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
        <b-col class="ms-2 col-sm-auto">
          <h2 class="fw-bold title-brand">Igor 2.3</h2>
        </b-col>
        <b-col class="d-flex align-items-center justify-content-center">
          <div v-if="isLoggedIn" class="w-100">
            <h5
                class="fw-bold text-center mb-0"
                id="motdMsg"
                :class="{
        'text-primary': true,
        'text-danger': motdFlag,
      }"
            >
              {{ motd }}
            </h5>
          </div>
        </b-col>
        <b-col class="me-3 col-sm-auto" align-v="bottom">
          <b-navbar class="justify-content-end">
            <b-navbar-nav class="align-items-center">
              <b-nav-text class="me-3">
                <b-form-checkbox
                  v-model="isDarkMode"
                  switch
                  class="mb-0 text-dark theme-toggle-control"
                  :aria-label="isDarkMode ? 'Disable dark mode' : 'Enable dark mode'"
                  @change="applyThemePreference"
                >
                  <b-icon
                    :icon="isDarkMode ? 'moon-stars-fill' : 'sun-fill'"
                    :variant="isDarkMode ? 'light' : 'warning'"
                    aria-hidden="true"
                  ></b-icon>
                  <span class="visually-hidden">
                    {{ isDarkMode ? "Disable dark mode" : "Enable dark mode" }}
                  </span>
                </b-form-checkbox>
              </b-nav-text>
              <b-nav-item-dropdown v-if="isLoggedIn" right>
                <!-- Using 'button-content' slot -->
                <template #button-content>
                  <span class="">
                    <b-icon-person-circle
                      aria-hidden="true"
                      variant="warning"
                      scale="1.5"
                      class="me-2"
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

const THEME_STORAGE_KEY = "igorweb-theme";

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
      isDarkMode: false,
    };
  },

  mounted() {
    this.initializeTheme();

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
    initializeTheme() {
      this.isDarkMode = localStorage.getItem(THEME_STORAGE_KEY) === "dark";
    },
    applyThemeClass() {
      //document.documentElement.classList.toggle("theme-dark", this.isDarkMode);
      //document.body.classList.toggle("theme-dark", this.isDarkMode);
      const currentTheme = document.documentElement.getAttribute('data-bs-theme');
      const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
      document.documentElement.setAttribute('data-bs-theme', newTheme);
    },
    applyThemePreference() {
      localStorage.setItem(THEME_STORAGE_KEY, this.isDarkMode ? "dark" : "light");
      this.applyThemeClass();
    },
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
        delete editData.fullName;
      }
      if (this.editUser.email === this.email) {
        delete editData.email;
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
