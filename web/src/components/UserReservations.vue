<template>
  <div>
    <!-- Modal for editing reservation -->
    <div>
      <b-modal ref="editModal" hide-footer title="Edit Reservation">
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
                          v-model="editResv.name"
                        ></b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="owner-group"
                        class="form-group col-6"
                        label-for="owner"
                        label="Owner"
                      >
                        <b-form-input
                          id="owner"
                          placeholder="Owner"
                          v-model="editResv.owner"
                        >
                        </b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="grp-group"
                        class="form-group col-8"
                        label-for="group"
                        label="Group"
                      >
                        <b-form-select
                          id="group"
                          v-model="editResv.group"
                          :options="groupNames"
                        >
                          <b-form-select-option value="none">
                            None</b-form-select-option
                          >
                        </b-form-select>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="description-group"
                        class="form-group col-8"
                        label-for="description"
                        label="Description"
                      >
                        <b-form-textarea
                          id="description"
                          placeholder="Description"
                          v-model="editResv.description"
                        ></b-form-textarea>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="saveResv(editResvId)"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="cancelEdit('editModal')"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </b-tab>
              <b-tab no-body title="Distro" title-link-class="text-dark">
                <div class="container">
                  <div class="row col-sm-6 form-group">
                    <label for="distro" class="col-form-label text-primary"
                      >Distro:</label
                    >
                    <b-form-select
                      id="distro"
                      v-model="editDistro.distro"
                      :options="eDistroNames"
                    ></b-form-select>
                  </div>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="saveDistro(editResvId)"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="cancelEdit"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </b-tab>
              <b-tab no-body title="Profile" title-link-class="text-dark">
                <div class="container">
                  <div class="row col-sm-6 form-group">
                    <label for="profile" class="col-form-label text-primary"
                      >Profile:</label
                    >
                    <b-form-select
                      id="profile"
                      v-model="editProfile.profile"
                      :options="eProfileNames"
                    ></b-form-select>
                  </div>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="saveProfile(editResvId)"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="cancelEdit"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </b-tab>
              <b-tab no-body title="Hosts" title-link-class="text-dark">
                <div class="container">
                  <div class="row col-sm-6 form-group">
                    <label for="hosts" class="col-form-label text-primary"
                      >Hosts:</label
                    >
                    <b-form-select
                      id="hosts"
                      :options="editHosts.hosts"
                      multiple
                      v-model="editHosts.hostsToRemove"
                    ></b-form-select>
                  </div>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-danger"
                      v-on:click="saveHosts(editResvId)"
                    >
                      Remove
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="cancelEdit"
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
    <!-- Modal for deleting reservation -->
    <div>
      <b-modal ref="deleteModal" hide-footer title="Delete Reservation">
        <div class="container">
          <div class="row">
            <p>Are you sure you want to delete this reservation?</p>
            <div class="modal-footer">
              <button
                type="button"
                v-on:click="deleteResv(deleteResvId)"
                class="btn btn-danger"
              >
                Delete
                <b-icon icon="trash" class="ml-1"></b-icon>
              </button>
              <button
                type="button"
                v-on:click="cancelDelete"
                class="btn btn-secondary"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      </b-modal>
    </div>
    <!-- Modal for extending reservation -->
    <div>
      <b-modal ref="extendModal" hide-footer title="Extend Reservation">
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
                  <div class="row col-sm-6 form-group">
                    <label for="extTime" class="col-form-label text-primary"
                      >Time:</label
                    >
                    <b-form-timepicker
                      locale="en"
                      id="extTime"
                      placeholder="Extension Time"
                      v-model="extendResv.extTime"
                    >
                    </b-form-timepicker>
                  </div>
                  <div class="row col-sm-6 form-group">
                    <label for="extTime" class="col-form-label text-primary"
                      >Date:</label
                    >
                    <b-form-datepicker
                      id="extDate"
                      v-model="extendResv.extDate"
                      :min="new Date()"
                    >
                    </b-form-datepicker>
                  </div>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-success"
                      v-on:click="extendMax(extendResvId)"
                    >
                      Extend Max
                    </button>
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="extendReservation(extendResvId)"
                    >
                      Save
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="cancelExtend"
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
    <!-- Power Cycle Modal -->
    <div>
      <b-modal ref="cycleModal" hide-footer title="Power Cycle Reservation">
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
                      v-model="cycleResvId"
                      disabled
                    ></b-form-input>
                  </div>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-success"
                      v-on:click="saveCycledResv('on')"
                    >
                      On
                    </button>
                    <button
                      type="button"
                      class="btn btn-danger"
                      v-on:click="saveCycledResv('off')"
                    >
                      Off
                    </button>
                    <button
                      type="button"
                      class="btn btn-warning"
                      v-on:click="saveCycledResv('cycle')"
                    >
                      Cycle
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="cancelCycle"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              </b-tab>
              <b-tab no-body title="Hosts" title-link-class="text-dark">
                <div class="container">
                  <div class="row col-sm-6 form-group">
                    <label for="hosts" class="col-form-label text-primary"
                      >Hosts:</label
                    >
                    <b-form-select
                      id="hosts"
                      :options="cycleHosts.hosts"
                      multiple
                      v-model="cycleHosts.hostsToCycle"
                    ></b-form-select>
                  </div>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-success"
                      v-on:click="saveCycledHosts('on')"
                    >
                      On
                    </button>
                    <button
                      type="button"
                      class="btn btn-danger"
                      v-on:click="saveCycledHosts('off')"
                    >
                      Off
                    </button>
                    <button
                      type="button"
                      class="btn btn-warning"
                      v-on:click="saveCycledHosts('cycle')"
                    >
                      Cycle
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary"
                      v-on:click="cancelCycle"
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
    <!-- Reservation Tab and table -->
    <div>
      <b-card no-body>
        <b-tabs card active-nav-item-class="font-weight-bold">
          <b-tab active no-body title=" My Reservations">
            <b-row class="mt-3">
            <b-col lg="6">
              <b-form-group
                label=""
                label-for="filter-input"
                
              >
                <b-input-group size="sm">
                  <b-form-input
                    id="filter-input"
                    v-model="filter"
                    type="search"
                    placeholder="Search"
                    class="form-control col-4 ml-3"
                  ></b-form-input>

                  <b-input-group-append>
                    <b-button :disabled="!filter" @click="filter = ''"
                      >Clear</b-button
                    >
                  </b-input-group-append>
                </b-input-group>
              </b-form-group>
            </b-col>
          </b-row>
            <b-row>
              <b-col>
                <b-table
                  hover
                  bordered
                  :items="rows"
                  :fields="fields"
                  :current-page="currentPage"
                  :per-page="perPage"
                  responsive="sm"
                  class="rtable pl-3 pr-3 pb-3"
                  show-empty
                  :filter="filter"
                :filter-included-fields="filterOn"
                @filtered="onFiltered"
                >
                  <template #empty="scope">
                    <h6 class="font-italic">{{ scope.emptyText }}</h6>
                  </template>
                  <template #cell(show_details)="row">
                    <div v-show="isUserReservation(row.item.name)">
                      <!-- Edit Reservation -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getEditId(row.item.name)"
                      >
                        <b-icon-pencil-fill
                          scale="0.8"
                          variant="primary"
                        ></b-icon-pencil-fill>
                      </b-button>
                      <!-- Extend Reservation -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getExtendId(row.item.name)"
                      >
                        <b-icon-clock-fill
                          scale="0.8"
                          variant="success"
                        ></b-icon-clock-fill>
                      </b-button>
                      <!-- Power Cycle Reservation -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getCycleId(row.item.name)"
                      >
                        <b-icon
                          icon="power"
                          class="rounded-circle bg-warning"
                          scale="0.7"
                          variant="white"
                        ></b-icon>
                      </b-button>
                      <!-- Delete Reservation -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getResvId(row.item.name)"
                      >
                        <b-icon-x-circle-fill
                          scale="0.8"
                          variant="danger"
                        ></b-icon-x-circle-fill>
                      </b-button>
                    </div>
                    <div v-show="!isUserReservation(row.item.name)">
                      <!-- Extend Reservation -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getExtendId(row.item.name)"
                      >
                        <b-icon-clock-fill
                          scale="0.8"
                          variant="success"
                        ></b-icon-clock-fill>
                      </b-button>
                      <!-- Power Cycle Reservation -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getCycleId(row.item.name)"
                      >
                        <b-icon
                          icon="power"
                          class="rounded-circle bg-warning"
                          scale="0.7"
                          variant="white"
                        ></b-icon>
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
  </div>
</template>

<script>
import axios from "axios";
import Vue from "vue";
import moment from "moment";
import SmartTable from "vuejs-smart-table";
Vue.use(SmartTable);
export default {
  name: "UserReservations",
  props: {
    msg: {
      type: String,
      value: "",
    },
  },
  data() {
    return {
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
          key: "end",
          thClass: "theader",
          thStyle: "font-weight: bold",
          formatter: (value) => {
            return moment(new Date(value * 1000)).format(
              "MMM[-]DD[-]YY[ ]h:mm a"
            );
          },
        },
        {
          sortable: true,
          key: "remainHours",
          label: "Hours Left",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
        {
          sortable: true,
          key: "hostRange",
          label: "Hosts",
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
      perPage: 10,
      totalPages: 0,
      totalRows: 1,
      filter: null,
      filterOn: [],
      deleteResvId: null,
      editResvId: null,
      extendResvId: null,
      cycleResvId: null,
      name: "",
      owner: "",
      description: "",
      group: "",
      editResv: {
        name: "",
        owner: "",
        group: "",
        description: "",
      },
      editDistro: {
        distro: "",
      },
      editProfile: {
        profile: "",
      },
      extendResv: {
        extTime: "",
        extDate: "",
        extDateTime: "",
      },
      editHosts: {
        hosts: [],
        hostsToRemove: [],
      },
      cycleHosts: {
        hosts: [],
        hostsToCycle: [],
      },
    };
  },
  mounted() {
    this.totalRows = this.$store.getters.associatedReservations.length;
  },
  computed: {
    userReservations() {
      return this.$store.getters.userReservations;
    },
    groupReservations() {
      return this.$store.getters.groupReservations;
    },
    associatedReservations() {
      return this.$store.getters.associatedReservations;
    },
    eDistroNames() {
      return this.$store.getters.eDistroNames;
    },
    eProfileNames() {
      return this.$store.getters.eProfileNames;
    },
    groupNames() {
      return this.$store.getters.groupNames;
    },
    rows() {
      if (!this.associatedReservations.length) {
        return [];
      }

      return this.associatedReservations
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
    onFiltered(filteredItems) {
      // Trigger pagination to update the number of buttons/pages due to filtering
      this.totalRows = filteredItems.length;
      this.currentPage = 1;
    },
    sort: function(col) {
      // if you click the same label twice
      if (this.currentSort == col) {
        this.currentSortDir = this.currentSortDir === "asc" ? "desc" : "asc";
      } else {
        this.currentSort = col;
      }
    },
    colValue: function(colName, colValue) {
      if (colName === "end") {
        let d = new Date(colValue * 1000);
        return moment(d).format("MMM[-]DD[-]YY[ ]h:mm a");
      }
      return colValue;
    },
    checkIfExpiring: function(colName, colValue) {
      if (colName === "remainHours") {
        if (colValue < 2) {
          return true;
        }
      }
      return false;
    },
    clearFilter() {
      this.searchText = "";
    },

    // Filter user owned reservations
    isUserReservation(id) {
      const index = this.userReservations.findIndex(
        (reservation) => reservation.name === id
      );
      if (~index) {
        return true;
      }
      return false;
    },

    // Methods for deleting reservation
    getResvId(id) {
      this.deleteResvId = id;
      this.$refs.deleteModal.show();
    },
    deleteResv(id) {
      let deleteResvUrl =
        this.$config.IGOR_API_BASE_URL + "/reservations/" + id;
      axios
        .delete(deleteResvUrl, { withCredentials: true })
        .then((response) => {
          //Find the index of the reservation for deletion
          const index = this.$store.getters.userReservations.findIndex(
            (reservation) => reservation.name === id
          );
          const indexAll = this.$store.getters.reservations.findIndex(
            (reservation) => reservation.name === id
          );
          const indexAssociated = this.$store.getters.associatedReservations.findIndex(
            (reservation) => reservation.name === id
          );
          this.updateHostStatus(index);
          this.$store.dispatch("deleteUserReservation", index);
          this.$store.dispatch("deleteAssociatedReservation", indexAssociated);
          this.$store.dispatch("deleteReservation", indexAll);
          this.deleteProfile();
          this.clearEditData();
          this.$refs.deleteModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    updateHostStatus(index) {
      // Free the Hosts when reservation is deleted
      let reservation = this.$store.getters.userReservations[index];
      let hosts = reservation.hosts;
      hosts.forEach((element) => {
        if (this.$store.getters.hostsResvPow.includes(element)) {
          let i = this.$store.getters.hostsResvPow.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsResvPow", payload);
        } else if (this.$store.getters.hostsResvDown.includes(element)) {
          let i = this.$store.getters.hostsResvDown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsResvDown", payload);
        } else if (this.$store.getters.hostsResvUnknown.includes(element)) {
          let i = this.$store.getters.hostsResvUnknown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsResvUnknown", payload);
        } else if (this.$store.getters.hostsInstErrPow.includes(element)) {
          let i = this.$store.getters.hostsInstErrPow.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsInstErrPow", payload);
        } else if (this.$store.getters.hostsInstErrDown.includes(element)) {
          let i = this.$store.getters.hostsInstErrDown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsInstErrDown", payload);
        } else if (this.$store.getters.hostsInstErrUnknown.includes(element)) {
          let i = this.$store.getters.hostsInstErrUnknown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsInstErrUnknown", payload);
        }
        this.$store.dispatch("addHostsForResv", element);
      });
    },

    deleteProfile() {
      let profileUrl = this.$config.IGOR_API_BASE_URL + "/profiles/";
      axios
        .get(profileUrl, { withCredentials: true })
        .then((response) => {
          let payload = response.data.data.profiles;
          this.$store.dispatch("insertProfiles", payload);
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    // Methods for editing reservation
    getEditId(id) {
      this.editResvId = id;
      const index = this.$store.getters.userReservations.findIndex(
        (reservation) => reservation.name === id
      ); // find the reservation index
      if (~index) {
        this.editResv.name = this.$store.getters.userReservations[index].name;
        this.editResv.owner = this.$store.getters.userReservations[index].owner;
        this.editResv.group = this.$store.getters.userReservations[index].group;
        this.editDistro.distro = this.$store.getters.userReservations[
          index
        ].distro;
        this.editProfile.profile = this.$store.getters.userReservations[
          index
        ].profile;
        this.editResv.description = this.$store.getters.userReservations[
          index
        ].description;
        this.editHosts.hosts = this.$store.getters.userReservations[
          index
        ].hosts;
      }
      this.name = this.$store.getters.userReservations[index].name;
      this.description = this.$store.getters.userReservations[
        index
      ].description;
      this.owner = this.$store.getters.userReservations[index].owner;
      this.group = this.$store.getters.userReservations[index].group;
      this.$refs.editModal.show();
    },
    isShown: function(option) {
      return !this.$store.getters.eDistroNames
        .map((item) => item.value)
        .includes(option.value);
    },

    // Save Reservation details
    saveResv(id) {
      let saveResvUrl = this.$config.IGOR_API_BASE_URL + "/reservations/" + id;
      let editData = {
        name: this.editResv.name,
        description: this.editResv.description,
        group: this.editResv.group,
        owner: this.editResv.owner,
      };
      // Sanity check
      if (this.editResv.name === this.name) {
        this.$delete(editData, "name");
      }
      if (this.editResv.description === this.description) {
        this.$delete(editData, "description");
      }
      if (this.editResv.owner === this.owner) {
        this.$delete(editData, "owner");
      }
      if (this.editResv.group === this.group) {
        this.$delete(editData, "group");
      }
      axios
        .patch(saveResvUrl, editData, { withCredentials: true })
        .then((response) => {
          const index = this.$store.getters.userReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~index) {
            let payload = {
              key: index,
              value: {
                name: this.editResv.name,
                description: this.editResv.description,
                owner: this.editResv.owner,
                group: this.editResv.group,
              },
            };
            this.$store.dispatch("saveResv", payload);
          }
          const indexAsc = this.$store.getters.associatedReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAsc) {
            let payload = {
              key: indexAsc,
              value: {
                name: this.editResv.name,
                description: this.editResv.description,
                owner: this.editResv.owner,
                group: this.editResv.group,
              },
            };
            this.$store.dispatch("saveAscResv", payload);
          }
          const indexAll = this.$store.getters.reservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAll) {
            let payload = {
              key: indexAll,
              value: {
                name: this.editResv.name,
                description: this.editResv.description,
                owner: this.editResv.owner,
                group: this.editResv.group,
              },
            };
            this.$store.dispatch("saveResvAll", payload);
          }
          alert("Reservation updated!");
          this.clearEditData();
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
          this.clearEditData();
        });
    },

    // Clear form data
    clearEditData() {
      this.deleteResvId = null;
      this.editResvId = null;
      this.extendResvId = null;
      this.cycleResvId = null;
      this.editResv = {
        name: "",
        owner: "",
        group: "",
        description: "",
      };
      this.editDistro = {
        distro: "",
      };
      this.editProfile = {
        profile: "",
      };
      this.extendResv = {
        extTime: "",
        extDate: "",
        extDateTime: "",
      };
      this.editHosts = {
        hosts: [],
        hostsToRemove: [],
      };
      this.cycleHosts = {
        hosts: [],
        hostsToCycle: [],
      };
    },

    // Save Distro details
    saveDistro(id) {
      let saveResvUrl = this.$config.IGOR_API_BASE_URL + "/reservations/" + id;
      let distroData = { distro: this.editDistro.distro };
      axios
        .patch(saveResvUrl, distroData, { withCredentials: true })
        .then((response) => {
          const index = this.$store.getters.userReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~index) {
            let payload = {
              key: index,
              value: {
                distro: this.editDistro.distro,
              },
            };
            this.$store.dispatch("saveDistro", payload);
          }
          const indexAsc = this.$store.getters.associatedReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAsc) {
            let payload = {
              key: index,
              value: {
                distro: this.editDistro.distro,
              },
            };
            this.$store.dispatch("saveDistroAsc", payload);
          }
          const indexAll = this.$store.getters.reservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAll) {
            let payload = {
              key: index,
              value: {
                distro: this.editDistro.distro,
              },
            };
            this.$store.dispatch("saveDistroAll", payload);
          }
          this.clearEditData();
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    // Save Profile details
    saveProfile(id) {
      let saveResvUrl = this.$config.IGOR_API_BASE_URL + "/reservations/" + id;
      let profileData = { profile: this.editProfile.profile };
      axios
        .patch(saveResvUrl, profileData, { withCredentials: true })
        .then((response) => {
          const index = this.$store.getters.userReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~index) {
            let payload = {
              key: index,
              value: {
                profile: this.editProfile.profile,
              },
            };
            this.$store.dispatch("saveResvNewProfile", payload);
          }
          const indexAsc = this.$store.getters.associatedReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAsc) {
            let payload = {
              key: index,
              value: {
                profile: this.editProfile.profile,
              },
            };
            this.$store.dispatch("saveProfileAsc", payload);
          }
          const indexAll = this.$store.getters.reservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAll) {
            let payload = {
              key: index,
              value: {
                profile: this.editProfile.profile,
              },
            };
            this.$store.dispatch("saveProfileAll", payload);
          }
          this.clearEditData();
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    // Update Hosts to be removed from the reservation
    saveHosts(id) {
      let saveResvUrl = this.$config.IGOR_API_BASE_URL + "/reservations/" + id;
      let hostsData = { drop: this.editHosts.hostsToRemove.toString() };
      axios
        .patch(saveResvUrl, hostsData, { withCredentials: true })
        .then((response) => {
          this.updateRemovedHost(this.editHosts.hostsToRemove);
          this.getHostRange(id);
          this.clearEditData();
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
    getHostRange(id) {
      let getResvUrl = this.$config.IGOR_API_BASE_URL + "/reservations";
      axios
        .get(getResvUrl, { withCredentials: true })
        .then((response) => {
          const i = response.data.data.reservations.findIndex(
            (reservation) => reservation.name === id
          );
          let resv = {};
          if (~i) {
            resv = response.data.data.reservations[i];
          }
          const index = this.$store.getters.userReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~index) {
            let payload = {
              key: index,
              value: {
                hostRange: resv.hostRange,
              },
            };
            this.$store.dispatch("saveHostRange", payload);
          }
          const indexAsc = this.$store.getters.asociatedReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAsc) {
            let payload = {
              key: index,
              value: {
                hostRange: resv.hostRange,
              },
            };
            this.$store.dispatch("saveHostRangeAsc", payload);
          }
          const indexAll = this.$store.getters.reservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAll) {
            let payload = {
              key: index,
              value: {
                hostRange: resv.hostRange,
              },
            };
            this.$store.dispatch("saveHostRangeAll", payload);
          }
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
    updateRemovedHost(hosts) {
      // Free the Hosts when hosts are removed from reservations
      hosts.forEach((element) => {
        if (this.$store.getters.hostsResvPow.includes(element)) {
          let i = this.$store.getters.hostsResvPow.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsResvPow", payload);
        } else if (this.$store.getters.hostsResvDown.includes(element)) {
          let i = this.$store.getters.hostsResvDown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsResvDown", payload);
        } else if (this.$store.getters.hostsResvUnknown.includes(element)) {
          let i = this.$store.getters.hostsResvUnknown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsResvUnknown", payload);
        } else if (this.$store.getters.hostsInstErrPow.includes(element)) {
          let i = this.$store.getters.hostsInstErrPow.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsInstErrPow", payload);
        } else if (this.$store.getters.hostsInstErrDown.includes(element)) {
          let i = this.$store.getters.hostsInstErrDown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsInstErrDown", payload);
        } else if (this.$store.getters.hostsInstErrUnknown.includes(element)) {
          let i = this.$store.getters.hostsInstErrUnknown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          this.$store.dispatch("removeHostsInstErrUnknown", payload);
        }
        this.$store.dispatch("addHostsForResv", element);
      });
    },

    // Methods for Power Cycling a Reservation
    getCycleId(id) {
      this.cycleResvId = id;
      const index = this.$store.getters.userReservations.findIndex(
        (reservation) => reservation.name === id
      ); // find the reservation index
      if (~index) {
        this.cycleHosts.hosts = this.$store.getters.userReservations[
          index
        ].hosts;
      }
      this.$refs.cycleModal.show();
    },

    // Update Hosts to be cycled from the reservation
    saveCycledHosts(command) {
      let saveCycledHostsUrl =
        this.$config.IGOR_API_BASE_URL + "/hosts-ctrl/power";
      let hostsData = {
        hosts: this.cycleHosts.hostsToCycle.toString(),
        cmd: command,
      };
      axios
        .patch(saveCycledHostsUrl, hostsData, { withCredentials: true })
        .then((response) => {
          this.clearEditData();
          this.$refs.cycleModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    saveCycledResv(command) {
      let saveCycledResvUrl =
        this.$config.IGOR_API_BASE_URL + "/hosts-ctrl/power";
      let resvData = {
        resName: this.cycleResvId,
        cmd: command,
      };
      axios
        .patch(saveCycledResvUrl, resvData, { withCredentials: true })
        .then((response) => {
          this.$refs.cycleModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    // Methods for extending reservation
    getExtendId(id) {
      this.extendResvId = id;
      this.$refs.extendModal.show();
    },
    extendReservation(id) {
      this.getExtendDateTime();
      let extendResvUrl =
        this.$config.IGOR_API_BASE_URL + "/reservations/" + id;
      let extendData = { extend: this.extendResv.extDateTime };
      axios
        .patch(extendResvUrl, extendData, { withCredentials: true })
        .then((response) => {
          this.getExtendedTime(id);
          alert("Reservation extended!");
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
      this.$refs.extendModal.hide();
    },
    getExtendDateTime() {
      let d = "";
      if (this.extendResv.extDate != "") {
        d = this.extendResv.extDate;
      } else {
        const today = new Date();
        d =
          today.getFullYear() +
          "-" +
          (today.getMonth() + 1) +
          "-" +
          today.getDate();
      }
      let t = "";
      if (this.extendResv.extTime != "") {
        t = this.extendResv.extTime;
      } else {
        let today = new Date();
        t = today.getHours() + ":" + today.getMinutes();
      }
      let dt = new Date(d + " " + t);
      this.extendResv.extDateTime = moment(dt).unix();
    },

    extendMax(id) {
      let extendResvMaxUrl =
        this.$config.IGOR_API_BASE_URL + "/reservations/" + id;
      let extendMax = { extendMax: true };
      axios
        .patch(extendResvMaxUrl, extendMax, { withCredentials: true })
        .then((response) => {
          this.getExtendedTime(id);
          alert("Reservation extended to Max Time");
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
      this.$refs.extendModal.hide();
    },

    getExtendedTime(id) {
      // Get the new extended time
      let resvUrl = this.$config.IGOR_API_BASE_URL + "/reservations";
      axios
        .get(resvUrl, { withCredentials: true })
        .then((response) => {
          let reservations = response.data.data.reservations;
          let remainHours = "";
          let end = "";
          const i = reservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~i) {
            remainHours = reservations[i].remainHours;
            end = reservations[i].end;
          }
          const index = this.$store.getters.userReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~index) {
            let payload = {
              key: index,
              value: {
                remainHours: remainHours,
                end: end,
              },
            };
            this.$store.dispatch("extendMax", payload);
          }

          const indexAsc = this.$store.getters.associatedReservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAsc) {
            let payload = {
              key: index,
              value: {
                remainHours: remainHours,
                end: end,
              },
            };
            this.$store.dispatch("extendMaxAsc", payload);
          }
          const indexAll = this.$store.getters.reservations.findIndex(
            (reservation) => reservation.name === id
          ); // find the reservation index
          if (~indexAll) {
            let payload = {
              key: indexAll,
              value: {
                remainHours: remainHours,
                end: end,
              },
            };
            this.$store.dispatch("extendMaxAll", payload);
          }
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
    cancelEdit() {
      this.clearEditData();
      this.$refs.editModal.hide();
    },
    cancelDelete(){
      this.clearEditData();
      this.$refs.deleteModal.hide();
    },
    cancelExtend(){
      this.clearEditData();
      this.$refs.extendModal.hide();
    },
    cancelCycle(){
      this.clearEditData();
      this.$refs.cycleModal.hide();
    },
  },
};
</script>
