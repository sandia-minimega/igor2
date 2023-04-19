<template>
  <div>
    <b-form @reset="onReset" @submit="onSubmit">
      <div>
        <b-card title="New Profile">
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
              id="distro-group"
              class="form-group col-sm-6"
              label-for="distro"
            >
              <template v-slot:label>
                Distro <span class="text-danger">*</span>
              </template>
              <b-form-select
                class="form-control col-sm-6"
                id="distro"
                v-model="form.distro"
                :options="distros"
                placeholder="Select Distro"
                required
              >
              </b-form-select>
            </b-form-group>
          </div>
          <div class="form-group row">
            <b-form-group
              id="kernelargs-group"
              class="form-group col-sm-6"
              label-for="kernelArgs"
              label="Kernel Args"
            >
              <b-form-input
                class="form-control col-sm-6"
                id="kernelArgs"
                v-model="form.kernelArgs"
                type="text"
                autocomplete="off"
                placeholder="Enter Kernel Args"
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
      <profile-table></profile-table>
    </div>
  </div>
</template>

<script>
import axios from "axios";
import ProfileTable from "./ProfileTable.vue";
export default {
  components: { ProfileTable },
  name: "CreateProfile",
  data() {
    return {
      form: {
        name: "",
        distros: "",
        kernelArgs: "",
        description: "",
      },
    };
  },
  computed: {
    distros() {
      return this.$store.getters.eDistroNames;
    },
  },
  methods: {
    onSubmit(event) {
      event.preventDefault();
      let payload = this.getPayload();
      let createProfileUrl = this.$config.IGOR_API_BASE_URL + "/profiles";
      axios
        .post(createProfileUrl, payload, { withCredentials: true })
        .then((response) => {
          payload = response.data.data.profile[0];
          this.$store.dispatch("insertNewProfile", payload);
          alert("Profile created successfully!");
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
      this.form.distro = "";
      this.form.kernelArgs = "";
    },
    getPayload() {
      let description = "";
      let kernelArgs = "";
      if (this.form.description != null) {
        description = this.form.description;
      }
      if (this.form.kernelArgs != null) {
        kernelArgs = this.form.kernelArgs;
      }
      let payload = {
        name: this.form.name,
        distro: this.form.distro,
        description: description,
        kernelArgs: kernelArgs,
      };
      return payload;
    },
  },
};
</script>
