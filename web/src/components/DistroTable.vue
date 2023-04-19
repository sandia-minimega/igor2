<template>
  <div>
    <!-- Show Details Modal -->
    <div>
      <b-modal ref="showDetailsModal" v-bind:title="showDistro.name">
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
                  <b-row class="mb-2">
                    <b-col sm="4" class="text-sm-right"><b>Owner:</b></b-col>
                    <b-col>{{ showDistro.owner }}</b-col>
                  </b-row>
                  <b-row>
                    <b-col sm="4" class="text-sm-right"
                      ><b>Description:</b></b-col
                    >
                    <b-col>{{ showDistro.description }}</b-col>
                  </b-row>

                  <b-row class="mb-2">
                    <b-col sm="4" class="text-sm-right"
                      ><b>Image Type:</b></b-col
                    >
                    <b-col>{{ showDistro.imageType }}</b-col>
                  </b-row>
                  <b-row>
                    <b-col sm="4" class="text-sm-right"
                      ><b>Kernel Args:</b></b-col
                    >
                    <b-col>{{ showDistro.kernelArgs }}</b-col>
                  </b-row>
                </div>
              </b-tab>
            </b-tabs>
          </b-card>
        </div>
      </b-modal>
    </div>

    <!-- Modal for editing distro -->
    <div>
      <b-modal ref="editModal" hide-footer title="Edit Distro">
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
                        class="form-group col-sm-6"
                        label-for="name"
                        label="Name"
                      >
                        <b-form-input
                          id="name"
                          placeholder="Name"
                          v-model="editDistro.name"
                        ></b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="kernelArgs-group"
                        class="form-group col-sm-6"
                        label-for="kernelArgs"
                        label="Kernel Args"
                      >
                        <b-form-input
                          id="kernelArgs"
                          placeholder="Kernel Args"
                          v-model="editDistro.kernelArgs"
                        ></b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="description-group"
                        class="form-group col-sm-6"
                        label-for="description"
                        label="Description"
                      >
                        <b-form-input
                          id="description"
                          placeholder="Description"
                          v-model="editDistro.description"
                        ></b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="distro-group"
                        class="form-group col-sm-6"
                        label-for="groups"
                      >
                        <template v-slot:label>
                          Groups
                          <b-button
                            class="align-center btn bg-transparent btn-outline-light text-dark buttonfocus"
                            @click="removeGroups"
                          >
                            <b-icon
                              icon="x-square-fill"
                              aria-hidden="true"
                              variant="danger"
                              scale="0.7"
                            ></b-icon>
                          </b-button>
                        </template>
                        <b-form-select
                          class="form-control col-sm-6"
                          id="groups"
                          v-model="remove"
                          type="text"
                          required
                          multiple
                          :options="editDistro.groups"
                        >
                        </b-form-select>
                      </b-form-group>
                      <b-form-group
                        id="distro-group1"
                        class="form-group col-sm-6"
                      >
                        <template v-slot:label>
                          Add Groups
                          <b-button
                            class="align-center btn bg-transparent btn-outline-light text-dark buttonfocus"
                            @click="addGroups"
                          >
                            <b-icon
                              icon="plus-square-fill"
                              aria-hidden="true"
                              variant="primary"
                              scale="0.7"
                            ></b-icon>
                          </b-button>
                        </template>
                        <b-form-select
                          class="form-control col-sm-6"
                          id="addGroups"
                          type="text"
                          multiple
                          :options="nonGroups"
                          v-model="add"
                        >
                        </b-form-select>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <div class="modal-footer">
                    <button
                      type="button"
                      class="btn btn-primary"
                      v-on:click="saveDistro(editDistroId)"
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
            </b-tabs>
          </b-card>
        </div>
      </b-modal>
    </div>
    <!-- Modal for deleting distro -->
    <div>
      <b-modal ref="deleteModal" hide-footer v-bind:title="deleteDistroId">
        <div class="container">
          <div class="row">
            <p>Are you sure you want to delete this distro?</p>
            <div class="modal-footer">
              <button
                type="button"
                v-on:click="deleteDistro(deleteDistroId)"
                class="btn btn-danger"
              >
                Delete
                <b-icon icon="trash" class="ml-1"></b-icon>
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
    <!-- Distro Table -->
    <div>
      <b-card no-body>
        <b-tabs card active-nav-item-class="font-weight-bold">
          <b-tab active no-body title="My Distros">
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
                >
                  <template #empty="scope">
                    <h6 class="font-italic">{{ scope.emptyText }}</h6>
                  </template>
                  <template #cell(show_details)="row">
                    <div v-show="userDistro(row.item.name)">
                      <!-- Edit Distro -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getEditId(row.item.name)"
                      >
                        <b-icon-pencil-fill
                          scale="0.7"
                          variant="primary"
                        ></b-icon-pencil-fill>
                      </b-button>
                      <!-- Delete Distro -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getDistroId(row.item.name)"
                      >
                        <b-icon-x-circle-fill
                          scale="0.7"
                          variant="danger"
                        ></b-icon-x-circle-fill>
                      </b-button>
                    </div>
                    <!-- Show Distro Details -->
                    <div v-show="!userDistro(row.item.name)">
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getShowId(row.item.name)"
                      >
                        <b-icon-arrows-fullscreen
                          scale="0.7"
                          variant="primary"
                        ></b-icon-arrows-fullscreen>
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
  </div>
</template>

<script>
import axios from "axios";
import $ from "jquery";
import Vue from "vue";
import SmartTable from "vuejs-smart-table";
Vue.use(SmartTable);
export default {
  name: "DistroTable",
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
        { label: "Description", shortcode: "description" },
        { label: "Groups", shortcode: "groups" },
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
          key: "groups",
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
      perPage: 10,
      totalPages: 0,
      deleteDistroId: null,
      editDistroId: null,
      editDistro: {
        name: "",
        description: "",
        kernelArgs: "",
        groups: [],
      },
      showDistroId: null,
      showDistro: {
        name: "",
        owner: "",
        kernelArgs: "",
        imageType: "",
      },
      name: "",
      description: "",
      kernelArgs: "",
      remove: [],
      add: [],
      nonGroups: [],
      newAdded: [],
      newRemoved: [],
      finalList: [],
    };
  },

  computed: {
    totalRows() {
      return this.distros.length;
    },
    distros() {
      return this.$store.getters.distros;
    },
    activeDistros() {
      return this.$store.getters.activeDistros;
    },
    ownerGroupNames() {
      return this.$store.getters.ownerGroupNames;
    },
    rows() {
      if (!this.distros.length) {
        return [];
      }

      return this.distros.filter((item) => {
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
      });
    },
  },

  methods: {
    // Filter user owned distros
    userDistro(id) {
      const index = this.distros.findIndex((distro) => distro.name === id);
      if (~index) {
        if (this.distros[index].owner === sessionStorage.getItem("username")) {
          if(this.activeDistros){
            if (!this.activeDistros[0].includes(id)) {
              // TODO: Fix the 2D array
              return true;
            }
          }
        }
      }
      return false;
    },
    // Methods for deleting distro
    getDistroId(id) {
      this.deleteDistroId = id;
      this.$refs.deleteModal.show();
    },
    deleteDistro(id) {
      let deleteDistroUrl = this.$config.IGOR_API_BASE_URL + "/distros/" + id;
      axios
        .delete(deleteDistroUrl, { withCredentials: true })
        .then((response) => {
          //Find the index of the distro for deletion
          const index = this.$store.getters.distros.findIndex(
            (distro) => distro.name === id
          );
          this.$store.dispatch("deleteDistro", index);
          const eDistroIndex = this.$store.getters.eDistroNames.findIndex(
            (distro) => distro === id
          );
          this.$store.dispatch("deleteEDistroNames", eDistroIndex);
          this.$refs.deleteModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    // Methods to Show Distro Details
    getShowId(id) {
      this.clearShowData();
      this.showDistroId = id;
      const index = this.$store.getters.distros.findIndex(
        (distro) => distro.name === id
      ); // find the distro index
      if (~index) {
        this.showDistro.name = this.$store.getters.distros[index].name;
        this.showDistro.owner = this.$store.getters.distros[index].owner;
        this.showDistro.imageType = this.$store.getters.distros[
          index
        ].image_type;
        this.showDistro.description = this.$store.getters.distros[
          index
        ].description;
        this.showDistro.kernelArgs = this.$store.getters.distros[
          index
        ].kernelArgs;
        this.$refs.showDetailsModal.show();
      }
    },

    // Clear form data
    clearShowData() {
      this.deleteDistroId = null;
      this.showDistroId = null;
      this.editDistroId = null;
      this.showDistro = {
        name: "",
        kernelArgs: "",
        description: "",
        owner: "",
        imageType: "",
      };
      this.$refs.editModal.hide();
    },

    // Methods for Editing Distro
    getEditId(id) {
      this.clearEditData();
      this.ownerGroupNames.forEach((element) => {
        this.nonGroups.push(element);
      });
      this.editDistroId = id;
      const index = this.$store.getters.distros.findIndex(
        (distro) => distro.name === id
      ); // find the distro index
      if (~index) {
        this.editDistro.name = this.$store.getters.distros[index].name;
        this.editDistro.description = this.$store.getters.distros[
          index
        ].description;
        this.editDistro.kernelArgs = this.$store.getters.distros[
          index
        ].kernelArgs;
        if (this.$store.getters.distros[index].groups != null) {
          this.$store.getters.distros[index].groups.forEach((element) => {
            this.editDistro.groups.push(element);
          });
        }
        this.name = this.$store.getters.distros[index].name;
        this.kernelArgs = this.$store.getters.distros[index].kernelArgs;
        this.description = this.$store.getters.distros[index].description;
        if (this.editDistro.groups) {
          this.editDistro.groups.forEach((element) => {
            const index = this.nonGroups.findIndex(
              (group) => group === element
            );
            if (~index) {
              this.nonGroups.splice(index, 1);
            }
          });
        }
        this.$refs.editModal.show();
      }
    },

    cancelEdit() {
      this.clearEditData();
      this.$refs.editModal.hide();
    },

    addGroups() {
      this.add.forEach((element) => {
        this.newAdded.push(element);
        this.editDistro.groups.push(element);
        const index = this.nonGroups.findIndex((group) => group === element);
        if (~index) {
          this.nonGroups.splice(index, 1);
        }
        const indexAdd = this.newRemoved.findIndex(
          (group) => group === element
        );
        if (~indexAdd) {
          this.newRemoved.splice(indexAdd, 1);
        }
      });
    },
    removeGroups() {
      this.remove.forEach((element) => {
        this.newRemoved.push(element);
        this.nonGroups.push(element);
        if (this.editDistro.groups != null) {
          const index = this.editDistro.groups.findIndex(
            (group) => group === element
          );
          if (~index) {
            this.editDistro.groups.splice(index, 1);
          }
        }
        const indexRemove = this.newAdded.findIndex(
          (group) => group === element
        );
        if (~indexRemove) {
          this.newAdded.splice(indexRemove, 1);
        }
      });
    },

    // Clear form data
    clearEditData() {
      this.deleteDistroId = null;
      this.editDistroId = null;
      this.editDistro = {
        name: "",
        kernelArgs: "",
        description: "",
        groups: [],
      };
      this.remove = [];
      this.add = [];
      this.nonGroups = [];
      this.newAdded = [];
      this.newRemoved = [];
      this.finalList = [];
      this.$refs.editModal.hide();
    },

    // Save Distro details
    saveDistro(id) {
      let saveDistroUrl = this.$config.IGOR_API_BASE_URL + "/distros/" + id;
      let editData = {
        name: this.editDistro.name,
        description: this.editDistro.description,
        kernelArgs: this.editDistro.kernelArgs,
        addGroup: this.newAdded,
        removeGroup: this.newRemoved,
      };

      // Sanity check
      let form_data = new FormData();
      if (this.editDistro.name !== this.name) {
        form_data.append("name", this.editDistro.name);
      }
      if (this.editDistro.kernelArgs !== this.kernelArgs) {
        form_data.append("kernelArgs", this.editDistro.kernelArgs);
      }
      if (this.editDistro.description !== this.description) {
        form_data.append("description", this.editDistro.description);
      }
      if (this.newAdded.length !== 0) {
        this.newAdded.forEach((element) => {
          form_data.append("addGroup", element);
        });
      }
      if (this.newRemoved.length !== 0) {
        this.newRemoved.forEach((element) => {
          form_data.append("removeGroup", element);
        });
      }

      axios
        .patch(saveDistroUrl, form_data, { withCredentials: true })
        .then((response) => {
          const index = this.$store.getters.distros.findIndex(
            (distro) => distro.name === id
          );
          if (~index) {
            let payload = {
              key: index,
              value: {
                name: this.editDistro.name,
                kernelArgs: this.editDistro.kernelArgs,
                description: this.editDistro.description,
                groups: this.editDistro.groups,
              },
            };
            this.$store.dispatch("saveUpdatedDistro", payload);
          }
          alert("Distro updated successfully!");
          this.clearEditData();
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          this.clearEditData();
          alert("Error: " + error.response.data.message);
        });
    },
  },
};
</script>
