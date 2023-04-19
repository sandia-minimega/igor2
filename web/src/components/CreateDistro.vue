<template>
  <div class="h-100">
    <!-- Modal for uploading K&I files -->
    <div>
      <b-modal ref="kiModal" hide-footer title="Kernel and Initrd files">
        <div class="container">
          <div class="row">
            <p>File upload in progress, do not navigate!</p>
          </div>
          <div class="modal-footer">
            <b-button variant="primary" disabled>
              <b-spinner small></b-spinner>
              Loading...
            </b-button>
            <!-- TODO Allow user to cancel the request if file upload takes longer -->
            <!-- <button
              type="button"
              variant="danger"
              v-on:click="cancelUpload"
              class="btn btn-danger"
            >
              Cancel
              <b-icon icon="x-circle" class="ml-1"></b-icon>
            </button> -->
          </div>
        </div>
      </b-modal>
    </div>
    <b-form @submit="onSubmit" @reset="onReset">
      <div>
        <b-card title="New Distro" class="w-100">
          <b-row>
            <b-col colspan="2">
              <b-form-group
                id="distro-type"
                class="form-group col-sm-4"
                label-for="distro"
              >
                <template v-slot:label>
                  Create from <span class="text-danger">*</span>
                </template>
                <b-form-select
                  id="distro"
                  v-model="form.distro"
                  :options="distros"
                  required
                  @change="next()"
                  class="form-control"
                ></b-form-select>
              </b-form-group>
            </b-col>
          </b-row>
        </b-card>
      </div>

      <div class="mt-3" v-if="showForm()">
        <b-card title="Distro Details" class="w-100">
          <b-row>
            <b-col colspan="2">
              <b-form-group
                class="form-group col-sm-4"
                id="edistro-group"
                v-show="eDistro"
              >
                <template v-slot:label>
                  Distro <span class="text-danger">*</span>
                </template>
                <b-form-select
                  class="form-control"
                  id="eDistroName"
                  v-model="form.eDistroName"
                  :options="eDistroNames"
                  required
                  @change="showDistroDetails()"
                ></b-form-select>
              </b-form-group>
              <b-form-group
                class="form-group col-sm-4"
                id="edistroimg-group"
                v-show="iRef"
              >
                <template v-slot:label>
                  Image <span class="text-danger">*</span>
                </template>
                <b-form-input
                  class="form-control"
                  id="eDistroImgName"
                  v-model="form.eDistroImgName"
                  required
                ></b-form-input>
              </b-form-group>
            </b-col>
          </b-row>
          <b-row>
            <b-col>
              <b-form-group
                class="form-group"
                id="kernelFile-group"
                v-show="ki"
              >
                <template v-slot:label>
                  Kernel File <span class="text-danger">*</span>
                </template>
                <b-form-file
                  id="kernelFile"
                  v-model="form.kernelFile"
                  :state="Boolean(form.kernelFile)"
                  placeholder="Choose/Drop a file"
                  drop-placeholder="Drop file here..."
                  @change="selectedFile($event)"
                ></b-form-file>
              </b-form-group>
            </b-col>
            <b-col>
              <b-form-group
                class="form-group"
                id="initrdFile-group"
                v-show="ki"
              >
                <template v-slot:label>
                  Initrd File <span class="text-danger">*</span>
                </template>
                <b-form-file
                  id="initrdFile"
                  v-model="form.initrdFile"
                  :state="Boolean(form.initrdFile)"
                  placeholder="Choose/Drop a file"
                  drop-placeholder="Drop file here..."
                  @change="selectedFile($event)"
                ></b-form-file>
              </b-form-group>
            </b-col>
          </b-row>
          <b-row>
            <b-col>
              <b-form-group id="name-group" class="form-group col-sm-6">
                <template v-slot:label>
                  Name <span class="text-danger">*</span>
                </template>
                <b-form-input
                  class="form-control"
                  id="name"
                  v-model="form.name"
                  type="text"
                  autocomplete="off"
                  placeholder="Enter name"
                  required
                ></b-form-input>
              </b-form-group>
            </b-col>
            <b-col>
              <b-form-group
                class="form-group col-sm-6"
                id="kernelargs-group"
                label="Kernel Args"
                label-for="kernelargs"
              >
                <b-form-input
                  class="form-control"
                  id="kernelargs"
                  v-model="form.kernelargs"
                  placeholder="Enter Kernel Arguments"
                ></b-form-input>
              </b-form-group>
            </b-col>
          </b-row>
          <b-row>
            <b-col>
              <b-form-group
                id="grpname-group"
                label="Group"
                label-for="grpname"
                class="form-group col-sm-6"
              >
                <b-form-select
                  class="form-control"
                  id="grpname"
                  v-model="form.grpname"
                  :options="ownerGroupNames"
                ></b-form-select>
              </b-form-group>
            </b-col>
            <b-col>
              <b-form-group id="public-group" class="form-group col-sm-6">
                <span class="align-top"
                  >Public
                  <b-form-checkbox
                    v-model="form.public"
                    switch
                    inline
                    class="align-top ml-2"
                    id="publicChkBox"
                  >
                  </b-form-checkbox>
                </span>
                <b-form-input
                  id="public"
                  placeholder="false"
                  class="form-control"
                  disabled
                  v-model="form.public"
                ></b-form-input>
              </b-form-group>
            </b-col>
          </b-row>
          <b-row>
            <b-col colspan="2">
              <b-form-group
                id="description-group"
                label="Description"
                label-for="description"
                class="form-group col-sm-10"
              >
                <b-form-textarea
                  class="form-control"
                  id="description"
                  v-model="form.description"
                  placeholder=""
                ></b-form-textarea>
              </b-form-group>
            </b-col>
          </b-row>
          <b-row>
            <b-col>
              <b-form-group>
                <b-button
                  variant="primary"
                  class="m-2"
                  type="submit"
                  :disabled="loading"
                  @click="onSubmit"
                  >Submit</b-button
                >
                <b-button
                  type="reset"
                  variant="outline-danger"
                  class="m-1"
                  @click="onReset">
                  Reset
                </b-button>
              </b-form-group>
            </b-col>
          </b-row>
        </b-card>
      </div>
    </b-form>
    <div class="mt-3">
      <distro-table></distro-table>
    </div>
  </div>
</template>

<script>
import axios from "axios";
import DistroTable from "./DistroTable.vue";
export default {
  components: { DistroTable },
  name: "CreateDistro",

  data() {
    return {
      loading: false,
      progress: 0,
      form: {
        eDistroName: "",
        eDistroImgName: "",
        kernelFile: null,
        initrdFile: null,
        name: "",
        kernelargs: "",
        grpname: "",
        distro: null,
        public: "false",
        description: "",
      },
      distros: [
        { text: "Select One", value: null },
        { text: "Distro", value: "edistro" },
        { text: "Image-Ref", value: "iref" },
        { text: "Distro-Img", value: "dimg" },
        { text: "K&I File Pair", value: "ki" },
      ],
      eDistro: false,
      eDImg: false,
      iRef: false,
      ki: false,
      distroType: false,
      show: true,
      step: 1,
      file: "",
      controller: new AbortController(),
      
    };
  },
  computed: {
    eDistroNames() {
      return this.$store.getters.eDistroNames;
    },
    eProfileNames() {
      return this.$store.getters.eProfileNames;
    },
    ownerGroupNames() {
      return this.$store.getters.ownerGroupNames;
    },
  },

  methods: {
    hideMsgModal() {
      this.$refs.kiModal.hide();
    },
    
    showDistroDetails() {
      if (!this.eDImg) {
        let id = this.form.eDistroName;
        const index = this.$store.getters.distros.findIndex(
          (distro) => distro.name === id
        ); // find the distro index
        if (~index) {
          this.form.description = this.$store.getters.distros[
            index
          ].description;
          this.form.grpname = this.$store.getters.distros[index].groups;
          this.form.kernelargs = this.$store.getters.distros[index].kernelArgs;
        }
      }
    },
    showForm() {
      if (this.distroType === true) {
        return true;
      } else {
        return false;
      }
    },

    getPayload() {
      let form_data = new FormData();
      form_data.append("name", this.form.name);
      form_data.append("description", this.form.description);
      if (this.eDistro) {
        form_data.append("copyDistro", this.form.eDistroName);
      }
      if (this.iRef) {
        form_data.append("imageRef", this.form.eDistroImgName);
      }
      if (this.ki) {
        form_data.append("kernelFile", this.form.kernelFile);
        form_data.append("initrdFile", this.form.initrdFile);
      }
      if (this.form.grpname != null) {
        if (this.form.grpname != "") {
          form_data.append("distroGroups", this.form.grpname);
        }
      }

      return form_data;
    },
    onReset(event) {
      //event.preventDefault();
      // Reset our form values
      this.form.eDistroImgName = "";
      this.form.kernelargs = "";
      this.form.name = "";
      this.form.grpname = "";
      this.form.description = "";
      this.form.eDistroName = "";
      this.form.kernelFile = null;
      this.form.initrdFile = null;
      this.form.public = "false";
      this.form.description = "";
      this.loading = false;
      // Trick to reset/clear native browser form validation state
      this.show = false;
      this.$nextTick(() => {
        this.show = true;
      });
    },

    onSubmit() {
      //event.preventDefault();
      let payload = this.getPayload();
      let createDistroUrl = this.$config.IGOR_API_BASE_URL + "/distros";
      this.loading = true;
      this.$refs.kiModal.show();
      axios
        .post(createDistroUrl, payload, { withCredentials: true },
          { "Content-Type" : "multipart/form-data" },
          { signal: this.controller.signal })
        .then((response) => {
          payload = response.data.data.distro[0];
          this.$store.dispatch("insertNewDistro", payload);
          this.$store.dispatch("addEDistroNames", this.form.name);
          this.hideMsgModal();
          this.ki = false;
          this.iRef = false;
          this.edistro = false;
          this.distroType = false;
          this.eDImg = false;
          this.form.distro = null;
          this.loading = false;
          this.onReset();
        })
        .catch(function(error) {
          alert("Error: " + error.response.data.message);
        })
        .then(() => {
          this.loading = false;
          this.hideMsgModal();
        });
        
    },

    next() {
      switch (this.form.distro) {
        case "edistro":
          this.eDistro = true;
          this.ki = false;
          this.iRef = false;
          this.distroType = true;
          this.eDImg = false;
          this.onReset();
          break;
        case "ki":
          this.ki = true;
          this.eDistro = false;
          this.iRef = false;
          this.distroType = true;
          this.eDImg = false;
          this.onReset();
          break;
        case "iref":
          this.ki = false;
          this.eDistro = false;
          this.iRef = true;
          this.distroType = true;
          this.eDImg = false;
          this.onReset();
          break;
        case "dimg":
          this.ki = false;
          this.eDistro = true;
          this.iRef = false;
          this.eDImg = true;
          this.distroType = true;
          this.onReset();
          break;
        default:
          this.ki = false;
          this.iRef = false;
          this.edistro = false;
          this.distroType = false;
          this.eDImg = false;
          this.onReset();
          break;
      }
    },
    selectedFile(event) {
      this.file = event.target.files[0];
    },
  },
};
</script>
