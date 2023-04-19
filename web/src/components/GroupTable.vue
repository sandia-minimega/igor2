<template>
  <div>
    <!-- Show Details Modal -->
    <div>
      <b-modal ref="showDetailsModal" v-bind:title="showGroup.name">
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
                title="Members"
                title-link-class="text-dark"
              >
                <div class="container">
                  <b-list-group>
                    <b-list-group-item
                      v-for="(member, index) in showGroup.fullNameMembers"
                      :key="index"
                    >
                      {{ member }}
                    </b-list-group-item>
                  </b-list-group>
                </div>
              </b-tab>
            </b-tabs>
          </b-card>
        </div>
      </b-modal>
    </div>
    <!-- Modal for editing group -->
    <div>
      <b-modal ref="editModal" hide-footer title="Edit Group">
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
                          v-model="editGroup.name"
                          class="form-control"
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
                          v-model="editGroup.owner"
                          class="form-control"
                        >
                        </b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="description-group"
                        class="form-group col-6"
                        label-for="description"
                        label="Description"
                      >
                        <b-form-input
                          id="description"
                          placeholder="Description"
                          v-model="editGroup.description"
                          class="form-control"
                        ></b-form-input>
                      </b-form-group>
                    </b-col>
                  </b-row>
                  <b-row>
                    <b-col>
                      <b-form-group
                        id="members-group"
                        class="form-group col-md-8"
                        label-for="members"
                      >
                        <template v-slot:label>
                          Members
                          <b-button
                            class="align-center btn bg-transparent btn-outline-light text-dark buttonfocus"
                            @click="removeMembers"
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
                          class="form-control col-md-8"
                          id="members"
                          v-model="remove"
                          type="text"
                          required
                          multiple
                          :options="editGroup.members"
                        >
                        </b-form-select>
                      </b-form-group>
                      <b-form-group
                        id="members-group1"
                        class="form-group col-md-8"
                      >
                        <template v-slot:label>
                          Add Members
                          <b-button
                            class="align-center btn bg-transparent btn-outline-light text-dark buttonfocus"
                            @click="addMembers"
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
                          class="form-control col-md-8"
                          id="addMembers"
                          type="text"
                          multiple
                          :options="nonMembers"
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
                      v-on:click="saveGroup(editGroupId)"
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
    <!-- Modal for deleting group -->
    <div>
      <b-modal ref="deleteModal" hide-footer v-bind:title="deleteGroupId">
        <div class="container">
          <div class="row">
            <p>Are you sure you want to delete this group?</p>
            <div class="modal-footer">
              <button
                type="button"
                v-on:click="deleteGroup(deleteGroupId)"
                class="btn btn-danger"
              >
                Delete
                <b-icon icon="trash" class="ml-1" scale="0.7"></b-icon>
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
    <!-- Group Table -->
    <div>
      <b-card no-body>
        <b-tabs card active-nav-item-class="font-weight-bold">
          <b-tab active no-body title="My Groups">
            <b-row>
              <b-col>
                <b-table
                  hover
                  bordered
                  :items="groups"
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
                    <div v-show="userGroup(row.item.name)">
                      <!-- Show Details -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getShowId(row.item.name)"
                      >
                        <b-icon-arrows-fullscreen
                          scale="0.7"
                          variant="primary"
                        ></b-icon-arrows-fullscreen>
                      </b-button>
                      <!-- Edit Group -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getEditId(row.item.name)"
                      >
                        <b-icon-pencil-fill
                          scale="0.7"
                          variant="primary"
                        ></b-icon-pencil-fill>
                      </b-button>
                      <!-- Delete Group -->
                      <b-button
                        class="btn bg-transparent btn-outline-light text-dark buttonfocus"
                        v-on:click="getGroupdId(row.item.name)"
                      >
                        <b-icon-x-circle-fill
                          scale="0.7"
                          variant="danger"
                        ></b-icon-x-circle-fill>
                      </b-button>
                    </div>
                    <!-- Show Members -->
                    <div v-show="!userGroup(row.item.name)">
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
export default {
  name: "GroupTable",
  props: {
    msg: {
      type: String,
      value: "",
    },
  },
  data() {
    return {
      numberOfCols: 3,
      fields: [
        {
          sortable: true,
          key: "name",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
        {
          sortable: true,
          key: "owner",
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
      deleteGroupId: null,
      editGroupId: null,
      editGroup: {
        name: "",
        owner: "",
        description: "",
        members: [],
      },
      remove: [],
      add: [],
      nonMembers: [],
      newAdded: [],
      newRemoved: [],
      finalList: [],
      showGroupId: null,
      showGroup: {
        name: "",
        members: [],
        fullNameMembers: [],
      },
    };
  },

  computed: {
    userDetails() {
      return this.$store.getters.userDetails;
    },
    groups() {
      return this.$store.getters.groups;
    },
    members() {
      let fullNames = [];
      let names = [];
      let members = {
        fullName: [],
        name: [],
      };
      this.$store.getters.userDetails.forEach((element) => {
        if (element.fullName) {
          fullNames.push(element.fullName);
        } else {
          fullNames.push(element.name);
        }
        names.push(element.name);
      });
      members.fullName = fullNames;
      members.name = names;
      return members;
    },
    fullNames(){
      let fullNames = [];
      this.$store.getters.userDetails.forEach((element) => {
        if (element.fullName) {
          fullNames.push(element.fullName);
        } else {
          fullNames.push(element.name);
        }
      });  
      const sortAlphaNum = (a, b) =>
        a.localeCompare(b, "en", { numeric: true });
      return fullNames.sort(sortAlphaNum);
    },
    totalRows() {
      return this.groups.length;
    },
    rows() {
      if (!this.groups.length) {
        return [];
      }

      return this.groups.filter((item) => {
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
    gridStyle() {
      return {
        gridTemplateColumns: `repeat(auto-fit, minmax(80px, 1fr))`,
        textAlign: `left`,
      };
    },
  },

  methods: {
    // Methods to Show Group Details
    getShowId(id) {
      this.clearShowData();
      this.showGroupId = id;
      const index = this.groups.findIndex((group) => group.name === id); // find the group index
      if (~index) {
        this.showGroup.name = this.groups[index].name;
        this.showGroup.members = this.groups[index].members;
        this.showGroup.members.forEach((element) => {
          this.showGroup.fullNameMembers.push(this.getFullName(element));
        });
        this.$refs.showDetailsModal.show();
      }
    },

    // Get Full Name for Group members
    getFullName(member) {
      const index = this.userDetails.findIndex((user) => user.name === member); // find the group index
      if (~index) {
        if (this.userDetails[index].fullName) {
          return this.userDetails[index].fullName;
        } else {
          return member;
        }
      }
    },
    // Clear form data
    clearShowData() {
      this.deleteGroupId = null;
      this.showGroupId = null;
      this.editGroupId = null;
      this.showGroup = {
        members: [],
        fullNameMembers: [],
      };
      this.$refs.showDetailsModal.hide();
    },

    // Filter user owned groups
    userGroup(id) {
      const index = this.groups.findIndex((group) => group.name === id);
      if (~index) {
        if (this.groups[index].owner === sessionStorage.getItem("username")) {
          return true;
        }
      }
      return false;
    },

    // Methods for deleting group
    getGroupdId(id) {
      this.deleteGroupId = id;
      this.$refs.deleteModal.show();
    },
    deleteGroup(id) {
      let deleteGroupUrl = this.$config.IGOR_API_BASE_URL + "/groups/" + id;
      axios
        .delete(deleteGroupUrl, { withCredentials: true })
        .then((response) => {
          //Find the index of the group for deletion
          const index = this.$store.getters.groups.findIndex(
            (group) => group.name === id
          );
          this.$store.dispatch("deleteGroup", index);
          this.$refs.deleteModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },

    // Methods for editing group
    getEditId(id) {
      this.fullNames.forEach((element) => {
        this.nonMembers.push(element);
      });
      this.editGroupId = id;
      const index = this.$store.getters.groups.findIndex(
        (group) => group.name === id
      ); // find the group index
      if (~index) {
        this.editGroup.name = this.$store.getters.groups[index].name;
        this.editGroup.owner = this.$store.getters.groups[index].owner;
        this.editGroup.description = this.$store.getters.groups[
          index
        ].description;
        this.name = this.$store.getters.groups[index].name;
        this.owner = this.$store.getters.groups[index].owner;
        this.description = this.$store.getters.groups[index].description;
        if (this.$store.getters.groups[index].members != null) {
          this.$store.getters.groups[index].members.forEach((element) => {
            this.editGroup.members.push(this.getFullName(element));
            this.finalList.push(this.getFullName(element));
          });
        }
        this.editGroup.members.forEach((element) => {
          const index = this.nonMembers.findIndex(
            (member) => member === element
          );
          if (~index) {
            this.nonMembers.splice(index, 1);
          }
        });
      }
      this.$refs.editModal.show();
    },
    addMembers() {
      this.add.forEach((element) => {
        this.newAdded.push(element);
        this.editGroup.members.push(element);
        const index = this.nonMembers.findIndex((member) => member === element);
        if (~index) {
          this.nonMembers.splice(index, 1);
        }
        const indexAdd = this.newRemoved.findIndex(
          (member) => member === element
        );
        if (~indexAdd) {
          this.newRemoved.splice(indexAdd, 1);
        }
      });
    },
    removeMembers() {
      this.remove.forEach((element) => {
        this.newRemoved.push(element);
        this.nonMembers.push(element);
        //Find the index of the group for editing
        const index = this.editGroup.members.findIndex(
          (member) => member === element
        );
        if (~index) {
          this.editGroup.members.splice(index, 1);
        }
        const indexRemove = this.newAdded.findIndex(
          (member) => member === element
        );
        if (~indexRemove) {
          this.newAdded.splice(indexRemove, 1);
        }
      });
    },

    // On cancel clear form data
    clearEditData() {
      this.deleteGroupId = null;
      this.editGroupId = null;
      this.editGroup = {
        name: "",
        owner: "",
        description: "",
        members: [],
      };
      this.remove = [];
      this.add = [];
      this.nonMembers = [];
      this.newAdded = [];
      this.newRemoved = [];
      this.finalList = [];
      this.$refs.editModal.hide();
    },

    // Save Group details
    saveGroup(id) {
      let saveGroupUrl = this.$config.IGOR_API_BASE_URL + "/groups/" + id;
      let addNames = [];
      this.newAdded.forEach((element) => {
        const index = this.members.fullName.findIndex(
          (member) => member === element
        );
        if (~index) {
          addNames.push(this.members.name[index]);
        }
      });
      let removeNames = [];
      this.newRemoved.forEach((element) => {
        const index = this.members.fullName.findIndex(
          (member) => member === element
        );
        if (~index) {
          removeNames.push(this.members.name[index]);
        }
      });
      let editData = {
        name: this.editGroup.name,
        description: this.editGroup.description,
        owner: this.editGroup.owner,
        add: addNames,
        remove: removeNames,
      };

      // Sanity check
      if (this.editGroup.name === this.name) {
        this.$delete(editData, "name");
      }
      if (this.editGroup.owner === this.owner) {
        this.$delete(editData, "owner");
      }
      if (this.editGroup.description === this.description) {
        this.$delete(editData, "description");
      }
      if (this.newAdded.length === 0) {
        this.$delete(editData, "add");
      }
      if (this.newRemoved.length === 0) {
        this.$delete(editData, "remove");
      }

      axios
        .patch(saveGroupUrl, editData, { withCredentials: true })
        .then((response) => {
          let userMembers = [];
          this.editGroup.members.forEach((element) => {
            const index = this.members.fullName.findIndex(
              (member) => member === element
            );
            if (~index) {
              userMembers.push(this.members.name[index]);
            }
          });
          const index = this.$store.getters.groups.findIndex(
            (group) => group.name === id
          ); // find the group index
          if (~index) {
            let payload = {
              key: index,
              value: {
                name: this.editGroup.name,
                owner: this.editGroup.owner,
                description: this.editGroup.description,
                members: userMembers,
              },
            };
            this.$store.dispatch("saveGroup", payload);
          }
          alert("Group updated successfully!");
          this.clearEditData();
          this.$refs.editModal.hide();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
    },
  },
};
</script>
