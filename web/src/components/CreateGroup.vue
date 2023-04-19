<template>
  <div class="h-100">
    <b-form @reset="onReset" @submit="onSubmit">
      <div>
        <b-card title="New Group">
          <div class="form-group row">
            <b-form-group
              id="name-group"
              class="form-group col-sm-6"
              label-for="name"
            >
              <template v-slot:label>
                Name <span class="text-danger">*</span>
              </template>
              <b-form-input
                class="form-control col-sm-6"
                id="name"
                v-model="form.name"
                type="text"
                autocomplete="off"
                placeholder="Enter Name"
                required
                autofocus
              >
              </b-form-input>
            </b-form-group>
            <b-form-group
              id="description-group"
              class="form-group col-sm-6"
              label-for="description"
              label="Description"
            >
              <b-form-textarea
                class="form-control col-sm-6"
                id="description"
                v-model="form.description"
                type="text"
                autocomplete="off"
                placeholder="Enter Description"
              >
              </b-form-textarea>
            </b-form-group>
          </div>
          <div class="form-group row">
            <b-form-group
              id="members-group"
              class="form-group col-sm-6"
              label-for="members"
            >
              <template v-slot:label>
                Add Members <span class="text-danger">*</span>
                <b-button
                  class="align-center btn bg-transparent btn-outline-light text-dark buttonfocus"
                  @click="addMembers"
                >
                  <b-icon
                    icon="arrow-right-square-fill"
                    aria-hidden="true"
                    variant="primary"
                    scale="0.7"
                  ></b-icon>
                </b-button>
              </template>
              <b-form-select
                class="form-control col-sm-6"
                id="members"
                v-model="form.members"
                required
                multiple
                :options="fullNames"
              >
              </b-form-select>
            </b-form-group>
            <b-form-group id="members-group1" class="form-group col-sm-6">
              <template v-slot:label>
                Members <span class="text-danger">*</span>
                <b-button
                  class="align-center btn bg-transparent btn-outline-light text-dark buttonfocus"
                  @click="removeMembers"
                >
                  <b-icon
                    icon="arrow-left-square-fill"
                    aria-hidden="true"
                    variant="danger"
                    scale="0.7"
                  ></b-icon>
                </b-button>
              </template>
              <b-form-select
                class="form-control col-sm-6"
                id="addMembers"
                multiple
                :options="form.newMembers"
                v-model="form.membersToRemove"
              >
              </b-form-select>
            </b-form-group>
          </div>
          <div class="form-group row pl-2">
            <b-form-group>
              <b-button type="submit" variant="primary" class="m-2"
                >Submit</b-button
              >
              <b-button type="reset" variant="outline-danger" class="m-1"
                >Reset</b-button
              >
            </b-form-group>
          </div>
        </b-card>
      </div>
    </b-form>
    <div class="mt-3">
      <group-table></group-table>
    </div>
  </div>
</template>

<style src="@vueform/multiselect/themes/default.css"></style>
<script>
import axios from "axios";
import GroupTable from "./GroupTable.vue";
export default {
  components: { GroupTable },
  name: "CreateGroup",

  data() {
    return {
      form: {
        name: "",
        members: [],
        newMembers: [],
        membersToRemove: [],
        description: "",
      },
      memberNames: [],
    };
  },

  computed: {
    allMembers() {
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
    }
  },
  methods: {
    addMembers() {
      this.form.members.forEach((element) => {
        this.form.newMembers.push(element);
      });
    },
    removeMembers() {
      this.form.membersToRemove.forEach((element) => {
        //Find the index of the member for deletion
        const index = this.form.newMembers.findIndex(
          (member) => member === element
        );
        if (~index) {
          this.form.newMembers.splice(index, 1);
        }
      });
    },
    onSubmit(event) {
      event.preventDefault();
      let payload = this.getPayload();
      let createGroupUrl = this.$config.IGOR_API_BASE_URL + "/groups";
      axios
        .post(createGroupUrl, payload, { withCredentials: true })
        .then((response) => {
          alert("Group created successfully!");
          this.$set(payload, "owner", sessionStorage.getItem("username"));
          this.$store.dispatch("insertNewGroup", payload);
          this.$store.dispatch("insertNewGroupName", payload.name);
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        });
      this.onReset(event);
    },
    onReset(event) {
      event.preventDefault();
      this.form.name = "";
      this.form.description = "";
      this.form.members = [];
      this.form.newMembers = [];
      this.form.membersToRemove = [];
    },
    getPayload() {
      let description = "";
      if (this.form.description != null) {
        description = this.form.description;
      }
      let memberNames = [];
      this.form.newMembers.forEach((element) => {
        const index = this.allMembers.fullName.findIndex(
          (member) => member === element
        );
        if (~index) {
          memberNames.push(this.allMembers.name[index]);
        }
      });

      let payload = {
        name: this.form.name,
        members: memberNames,
        description: description,
      };
      return payload;
    },
  },
};
</script>
