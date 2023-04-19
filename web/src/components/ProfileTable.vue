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
              <b-tab
                active
                no-body
                title="Details"
                title-link-class="text-dark"
              >
                <div class="container">
                  <div class="row col-sm-6">
                    <label for="name" class="col-form-label text-primary"
                      >Name:</label
                    >
                    <b-form-input
                      id="name"
                      placeholder="Name"
                      v-model="editProfile.name"
                    ></b-form-input>
                  </div>
                  <div class="row col-sm-6">
                    <label for="kernelArgs" class="col-form-label text-primary"
                      >Kernel Args:</label
                    >
                    <b-form-input
                      id="kernelArgs"
                      placeholder="Kernel Args"
                      v-model="editProfile.kernelArgs"
                    >
                    </b-form-input>
                  </div>
                  <div class="row form-group col-sm-6">
                    <label for="description" class="col-form-label text-primary"
                      >Description:</label
                    >
                    <b-form-input
                      id="description"
                      placeholder="Description"
                      v-model="editProfile.description"
                    ></b-form-input>
                  </div>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="saveProfile(editProfileId)"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="$refs.editModal.hide()"
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
    <!-- Modal for deleting profile -->
    <div>
      <b-modal ref="deleteModal" hide-footer v-bind:title="deleteProfileId">
        <div class="container">
          <div class="row">
            <p>Are you sure you want to delete this profile?</p>
            <div class="modal-footer">
              <button
                type="button"
                v-on:click="deleteProfile(deleteProfileId)"
                class="btn btn-danger"
              >
                Delete
                <b-icon icon="trash" scale="0.7" class="ml-1"></b-icon>
              </button>
              <button
                type="button"
                v-on:click="$refs.deleteModal.hide()"
                class="btn btn-secondary"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      </b-modal>
    </div>
    <!-- Profile Table -->
    <b-card no-body>
      <b-tabs card active-nav-item-class="font-weight-bold">
        <b-tab active no-body title=" My Profiles">
          <b-row>
            <b-col>
              <b-table
                hover
                bordered
                :items="profiles"
                :fields="fields"
                :current-page="currentPage"
                :per-page="perPage"
                responsive="sm"
                class="rtable pl-3 pr-3 pb-3"
                show-empty
              >
                <template #empty="scope">
                  <h6 class="font-italic">{{ scope.emptyText }}</h6>
                </template>
                <template #cell(show_details)="row">
                  <div v-show="userProfile(row.item.name)">
                    <!-- Edit Profile -->
                    <b-button
                      class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                      v-on:click="getEditId(row.item.name)"
                    >
                      <b-icon-pencil-fill
                        scale="0.7"
                        variant="primary"
                      ></b-icon-pencil-fill>
                    </b-button>
                    <!-- Delete Profile -->
                    <b-button
                      class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                      v-on:click="getProfileId(row.item.name)"
                    >
                      <b-icon-x-circle-fill
                        scale="0.7"
                        variant="danger"
                      ></b-icon-x-circle-fill>
                    </b-button>
                  </div>
                </template>
              </b-table>
            </b-col>
          </b-row>
          <b-row>
            <b-col>
              <b-pagination
                :total-rows="totalRows"
                v-model="currentPage"
                :current-page="currentPage"
                :per-page="perPage"
                class="my-0"
                size="sm"
                align="fill"
                aria-controls="my-table"
              ></b-pagination>
            </b-col>
          </b-row>
        </b-tab>
      </b-tabs>
    </b-card>
  </div>
</template>

<script>
import Vue from "vue";
import SmartTable from "vuejs-smart-table";
Vue.use(SmartTable);
import axios from "axios";
export default {
  name: "ProfileTable",
  props: {
    msg: {
      type: String,
      value: "",
    },
  },
  data() {
    return {
      columns: [
        { label: "Name", shortcode: "name" },
        { label: "Distro", shortcode: "distro" },
        { label: "Kernel Args", shortcode: "kernelArgs" },
        { label: "Description", shortcode: "description" },
      ],
      fields: [
        {
          sortable: true,
          key: "name",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
        {
          sortable: true,
          key: "distro",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
        {
          sortable: true,
          key: "kernelArgs",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
        {
          sortable: true,
          key: "description",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
        {
          key: "show_details",
          label: "",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
      ],
      search: null,
      column: null,
      currentSort: "pn",
      currentSortDir: "asc",
      currentPage: 1,
      totalPages: 0,
      perPage: 10,
      deleteProfileId: null,
      editProfileId: null,
      editProfile: {
        name: "",
        kernelArgs: "",
        description: "",
      },
    };
  },

  computed: {
    profiles() {
      return this.$store.getters.profiles;
    },
    totalRows() {
      return this.profiles.length;
    },
    activeProfiles() {
      return this.$store.getters.activeProfiles;
    },
    rows() {
      if (!this.profiles.length) {
        return [];
      }

      return this.profiles
        .filter((item) => {
          let props =
            this.search && this.column
              ? [item[this.column]]
              : Object.values(item);

          return props.some(
            (prop) =>
              !this.search ||
              (typeof prop === "string"
                ? prop.includes(this.search)
                : prop.toString(10).includes(this.search))
          );
        })
        .sort((a, b) => {
          if (this.currentSortDir === "asc") {
            return a[this.currentSort] >= b[this.currentSort];
          }
          return a[this.currentSort] <= b[this.currentSort];
        });
    },
  },

  methods: {
    // Methods for deleting profile
    getProfileId(id) {
      this.deleteProfileId = id;
      this.$refs.deleteModal.show();
    },
    deleteProfile(id) {
      let deleteProfileUrl = this.$config.IGOR_API_BASE_URL + "/profiles/" + id;
      axios
        .delete(deleteProfileUrl, { withCredentials: true })
        .then((response) => {
          //Find the index of the profile for deletion
          const index = this.$store.getters.profiles.findIndex(
            (profile) => profile.name === id
          );
          this.$store.dispatch("deleteProfile", index);
          this.$refs.deleteModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
    // Filter user owned groups
    userProfile(id) {
      return !this.activeProfiles[0].includes(id); // TODO: Fix the 2D array
    },

    // Methods for Editing Profile
    getEditId(id) {
      this.editProfileId = id;
      const index = this.$store.getters.profiles.findIndex(
        (profile) => profile.name === id
      ); // find the profile index
      if (~index) {
        this.editProfile.name = this.$store.getters.profiles[index].name;
        this.editProfile.kernelArgs = this.$store.getters.profiles[
          index
        ].kernelArgs;
        this.editProfile.description = this.$store.getters.profiles[
          index
        ].description;
      }
      this.$refs.editModal.show();
    },

    // Save Profile details
    saveProfile(id) {
      let saveProfileUrl = this.$config.IGOR_API_BASE_URL + "/profiles/" + id;
      let editData = this.editProfile;
      axios
        .patch(saveProfileUrl, editData, { withCredentials: true })
        .then((response) => {
          const index = this.$store.getters.profiles.findIndex(
            (profile) => profile.name === id
          ); // find the profile index
          if (~index) {
            let payload = {
              key: index,
              value: {
                name: this.editProfile.name,
                kernelArgs: this.editProfile.kernelArgs,
                description: this.editProfile.description,
              },
            };
            this.$store.dispatch("saveProfile", payload);
          }
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
  },
};
</script>
