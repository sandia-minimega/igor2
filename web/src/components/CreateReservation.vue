<template>
  <div class="h-100">
    <b-form @submit="onSubmit" @reset="onReset">
      <div>
        <b-card title="New Reservation" class="w-100">
          <b-row>
            <b-col>
              <b-form-group
                  id="name-group"
                  label=""
                  label-for="name"
                  class="form-group col-sm-6"
                >
                  <template v-slot:label>
                    Name <span class="text-danger">*</span>
                  </template>
                  <b-form-input
                    id="name"
                    v-model="form.name"
                    type="text"
                    autocomplete="off"
                    placeholder="Enter name"
                    required
                    class="form-control"
                    autofocus
                    inline
                ></b-form-input>
              </b-form-group>
            </b-col>
            <b-col>
              <div>
                <b-form-group label="" v-slot="{ ariaDescribedby }" class="form-group col-sm-6">
                  <b-form-radio v-model="form.reservation" @change="next()" :aria-describedby="ariaDescribedby" value="eProfile">
                    Profile <span class="text-danger">*</span> 
                  </b-form-radio>
                  <b-form-radio v-model="form.reservation" @change="next()" :aria-describedby="ariaDescribedby" value="eDistro">
                    Distro</b-form-radio>
                  <b-form-select
                    id="eDistroProfileName"
                    v-model="form.eDistroProfileName"
                    :options="eDistroProfileNames"
                    @change="selectedProfileDistro()"
                    class="form-control mt-2"
                  ></b-form-select>
                </b-form-group>
              </div>
            </b-col>
          </b-row>
        </b-card>
      </div>

      <div v-if="showForm()" class="mt-3">
        <b-card title="Hosts" sub-title="(Use count or select hosts to reserve)" class="w-100">
          <b-row>
          <b-col>
            <b-form-group
              id="nodeCount-group"
              label="Count"
              label-for="nodeCount"
              class="form-group col-sm-6"
            >
              <b-form-input
                id="nodeCount"
                v-model.lazy="nodeCountFromGrid"
                type="number"
                placeholder=""
                min="0"
                class="form-control"
                :disabled="hostAllowSelect == true"
              ></b-form-input>
              
            </b-form-group>
          </b-col>
          <b-col>
            <b-form-group
              id="nodeOption-group"
              label=""
              label-for="nodeOption"
              class="form-group col-sm-2"
            >
              <b-form-checkbox
                id="nodeOption"
                v-model="hostSelect"
                @change="enableHostSelect"
                switch
              >
              OR
              </b-form-checkbox>
            </b-form-group>
          </b-col>
          <b-col>
            <b-form-group
              id="nodeList-group"
              class="form-group col-sm-10"
            >
            <template v-slot:label>
                Select
                <b-button
                  class="align-center btn bg-transparent btn-outline-light text-dark buttonfocus"
                  @click="clearHosts"
                >
                  <b-icon
                    icon="x-square-fill"
                    aria-hidden="true"
                    variant="danger"
                    scale="0.7"
                  ></b-icon>
                </b-button>
              </template>
              <b-form-textarea
                id="nodeListT"
                v-model.lazy="hostsFromGrid"
                class="form-control"
                v-if="!nodeListText"
                :disabled="hostAllowSelect == false"
              >
              </b-form-textarea>
            </b-form-group>
          </b-col>
          </b-row>
        </b-card>
        <b-card title="Time Start/End" :sub-title="minResHours" class="w-100 mt-3">
          <b-row>
            <b-col>
            <b-form-group
              id="startDate-group"
              label=""
              label-for="startDate"
              class="form-group col-sm-6"
            >
            <b-form-checkbox
              switch
              inline
              class="align-top ml-2"
              v-model="dateTimeChecked"
              @change="dateTimeToggle"
            >
            Start Time
            
            </b-form-checkbox>
              <b-form-datepicker
                id="startDate"
                v-model="form.startDate"
                v-if="dateTimeText"
                :min="new Date()"
                class="form-control"
              ></b-form-datepicker>
              <b-form-timepicker
                id="startTime"
                v-model="form.startTime"
                v-if="dateTimeText"
                class="form-control"
                now-button
                label-now-button="Current Time"
                reset-button
                no-close-button
              ></b-form-timepicker>
              <b-form-input
                id="dateTimeT"
                v-model="form.dateTimeTextValue"
                v-if="!dateTimeText"
                type="text"
                autocomplete="off"
                class="form-control"
                :placeholder="resvStartTime"
                disabled
              >
              </b-form-input>
            </b-form-group>
            </b-col>
            <b-col>
            <b-form-group
              id="duration-group"
              label=""
              label-for="duration"
              class="form-group col-sm-6"
            >
              <b-form-checkbox
                  switch
                  inline
                  class="align-top ml-2"
                  v-model="endDateTimeChecked"
                  @change="endDateTimeToggle"
                >End Time
              </b-form-checkbox>
              <b-form-datepicker
                id="endDate"
                v-model="form.endDate"
                v-if="endDateTimeText"
                :min="new Date()"
                class="form-control"
              ></b-form-datepicker>
              <b-form-timepicker
                id="endTime"
                v-model="form.endTime"
                v-if="endDateTimeText"
                class="form-control"
                now-button
                label-now-button="Current Time"
                reset-button
                no-close-button
              ></b-form-timepicker>
              <b-form-input
                id="duration"
                v-model="form.endDateTimeTextValue"
                v-if="!endDateTimeText"
                type="text"
                autocomplete="off"
                class="form-control"
                :placeholder="resvEndTime"
                disabled
              ></b-form-input>
            </b-form-group>
          </b-col>
          </b-row>
        </b-card>
        <b-card title="" class="w-100 mt-3">
          <b-row>
            <b-col>
            <b-form-group
              id="grpname-group"
              label="Group"
              label-for="grpname"
              class="form-group col-sm-6"
            >
              <b-form-select
                id="grpname"
                v-model="form.grpname"
                :options="groupNames"
                class="form-control"
              ></b-form-select>
            </b-form-group>
            </b-col>
            <b-col>
            <b-form-group
              id="vlan-group"
              label=""
              label-for="vlan"
              class="form-group col-sm-6"
            >
              <b-form-checkbox
                  switch
                  inline
                  class="align-top ml-2"
                  v-model="vlanChecked"
                  @change="vlanToggle"
                >
                Vlan
              </b-form-checkbox>
              <b-form-select
                id="vlan"
                v-model="form.vlan"
                autocomplete="off"                
                class="form-control"
                v-show="!vlanValue"
                :options="reservationNames"
              ></b-form-select>
              <b-input
              class="form-control mt-2"
              type="text"
              v-model="form.vlanVal"
              v-show="vlanValue"
              :placeholder="minmaxVal()">
              </b-input>
            </b-form-group>
          </b-col>
          </b-row>
        </b-card>
        <b-card title="" class="w-100 mt-3">
          <b-row>
            <b-col colspan="2">
            <b-form-group
              id="description-group"
              label="Description"
              label-for="description"
              class="form-group col-sm-10"
            >
              <b-form-textarea
                id="description"
                v-model="form.description"
                type="text"
                autocomplete="off"
                placeholder="Description"
                class="form-control"
              ></b-form-textarea>
            </b-form-group>
            </b-col>
          </b-row>
        </b-card>
          <b-row class="mt-3">
            <b-col colspan="2">
              <b-form-group>
                <b-form-checkbox v-model="powerCycleStatus" @change="doNotCycle">
                  <p class="font-weight-bold">Do not Power Cycle on start</p>
                </b-form-checkbox>
              </b-form-group>
            </b-col>
          </b-row>
          <b-row>
            <b-col colspan="2">
            <b-form-group>
              <b-button type="submit" variant="primary" class="m-2"
                >Submit</b-button
              >
              <b-button type="reset" variant="outline-danger" class="m-1"
                >Reset</b-button
              >
            </b-form-group>
            </b-col>
          </b-row>
      </div>
    </b-form>
    <div class="mt-3">
      <user-reservations></user-reservations>
    </div>
  </div>
</template>

<script>
import axios from "axios";
import moment from "moment";
import UserReservations from "./UserReservations.vue";
export default {
  components: { UserReservations },
  name: "CreateReservation",
  data() {
    return {
      form: {
        eDistroName: "",
        eProfileName: "",
        eDistroProfileName: "",
        name: "",
        startDate: "",
        startTime: "",
        endDate: "",
        endTime: "",
        vlan: "",
        vlanVal: "",
        nodeList: [],
        nodeListTextValue: "",
        nodes: "",
        nodeCount: 0,
        owner: "",
        description: "",
        grpname: "",
        reservation: "",
        start: "",
        dateTimeTextValue: "",
        startDT: "",
        end: "",
        endDateTimeTextValue: "",
        endDT: "",
      },
      powerCycleStatus: false,
      noCycle: false,
      eDistro: false,
      eProfile: false,
      eDistroProfileNames: [],
      show: true,
      step: 1,
      file: "",
      reservFlag: false,
      validateFlag: true,
      imageSelected: false,
      nodeListChecked: false,
      hostSelect: false,
      hostAllowSelect: false,
      nodeListText: false,
      dateTimeText: false,
      dateTimeChecked: false,
      endDateTimeText: false,
      endDateTimeChecked: false,
      vlanChecked: false,
      timestamp: "",
      futureResv: false,
      selected: '',
      vlanValue: false,
      resvStartTime: "",
      resvEndTime: "",
    };
  },
  
  computed: {
       
    eDistroNames() {
      return this.$store.getters.eDistroNames;
    },
    eProfileNames() {
      return this.$store.getters.eProfileNames;
    },
    groups() {
      return this.$store.getters.groups;
    },
    groupNames() {
      return this.$store.getters.groupNames;
    },
    hostNames() {
      return this.$store.getters.hostNames;
    },
    hostsForResv() {
      return this.$store.getters.hostsForResv;
    },
    currentDate() {
      return (
        "Reservation starts at: " +
        moment()
          .add(5, "minutes")
          .format("MM[/]DD[/]YYYY[ ]hh:mm")
      );
    },
    defaultReserveMinutes() {
      return this.$store.getters.defaultReserveMinutes;
    },
    minResHours() {
      if(!this.dateTimeText){
        return "(Default starts now and ends in " + Math.floor(this.defaultReserveMinutes / 60) + " hour/s. Use slider for custom date/time.)";
      }
      else{
        return "";
      }
    },
    associatedReservations() {
      return this.$store.getters.associatedReservations;
    },
    reservationNames() {
      let reservationNames = []
      this.associatedReservations.forEach(element => {
        reservationNames.push(element.name);  
      });
      return reservationNames;
    },
    vlanMin() {
      return this.$store.getters.vlanMin;
    },
    vlanMax() {
      return this.$store.getters.vlanMax;
    },
    selectedHosts() {
      return this.$store.getters.selectedHosts;
    },
    hostsFromGrid: {
      get () {
        if(this.$store.getters.selectedHosts != []) {
          return this.$store.getters.selectedHosts.join();
        }
        else {
          return "";
        }
      },
      set (value) {
        if(value != "") {
          if(value.includes(",")) {
            this.$store.dispatch('selectedResvHosts', value.split(","));
          }
          else if(value.includes("-")) {
            if(value.includes("]")){
              let parseHostList = value.substr(value.indexOf("[")+1);
              let hostRange = parseHostList.substr(0, parseHostList.indexOf("]"));
              let prefix = this.$store.getters.clusterPrefix;
              let str = hostRange.split("-");
              let hostList = [];
              for(let count = str[0]; count < parseInt(str[1])+1; count++){
                hostList.push(prefix+count);
              }
              this.$store.dispatch('selectedResvHosts', hostList);
            }
          }
          else {
            this.$store.dispatch('selectedResvHosts', value.split());
          }
        }
        else {
          this.$store.dispatch('selectedResvHosts', []);
        }
      }  
    },
    nodeCountFromGrid: {
      get () {
        if(this.$store.getters.selectedHosts.length != 0){
          return this.$store.getters.selectedHosts.length;
        }
        else {
          return this.$store.getters.selectedHostsCount;
        }
      },
      set (value) {
        if(this.$store.getters.selectedHosts.length != 0){
          this.$store.dispatch('selectedResvHostsCount', this.$store.getters.selectedHosts.length);
        }
        else {
          this.$store.dispatch('selectedResvHostsCount', value);  
        }
      }
    },    
    hostsFromGridSelected() {
      if(this.$store.getters.selectedHosts != []) {
        return true;
      }
      else {
        return false;
      }
    },
  },
  mounted() {
    this.currentTime();
    this.minDurationTime();
  },
  methods: {
    clearHosts(){
      this.$store.dispatch('selectedResvHosts', []);
    },
    currentTime(){
      setInterval(() => this.getCurrentTime(), 5000);
    },
    getCurrentTime(){
      this.resvStartTime = moment().format('MM/DD/YY hh:mm');
    },
    minDurationTime(){
      setInterval(() => this.getMinDurationTime(), 5000);  
    },
    getMinDurationTime(){
      this.resvEndTime = moment().add(this.defaultReserveMinutes, "minutes")
                        .format('MM/DD/YY hh:mm');
    },
    vlanToggle() {
      if (this.vlanChecked == true) {
        this.vlanValue = true;
        this.form.vlan = "";
        this.form.vlanVal = "";
      } else {
        this.vlanValue = false;
        this.form.vlan = "";
        this.form.vlanVal = "";
      }
    },

    dateTimeToggle() {
      if (this.dateTimeChecked == true) {
        this.dateTimeText = true;
        this.form.startDate = "";
        this.form.startTime = "";
        this.form.start = "";
        this.form.startDT = "";
      } else {
        this.dateTimeText = false;
        this.form.startDate = "";
        this.form.startTime = "";
        this.form.start = "";
        this.form.startDT = "";
      }
    },

    endDateTimeToggle() {
      if (this.endDateTimeChecked == true) {
        this.endDateTimeText = true;
        this.form.endDate = "";
        this.form.endTime = "";
        this.form.end = "";
        this.form.endDT = "";
      } else {
        this.endDateTimeText = false;
        this.form.endDate = "";
        this.form.endTime = "";
        this.form.end = "";
        this.form.endDT = "";
      }
    },

    enableHostSelect() {
      if (this.hostSelect == true) {
        this.hostAllowSelect = true;
      } else {
        this.hostAllowSelect = false;
      }
    },

    nodeListToggle() {
      if (this.nodeListChecked == true) {
        this.nodeListText = true;
        this.form.nodeList = [];
        this.form.nodeListTextValue = "";
        this.form.nodes = "";
      } else {
        this.nodeListText = false;
        this.form.nodeList = [];
        this.form.nodeListTextValue = "";
        this.form.nodes = "";
      }
    },

    showForm() {
      if (this.imageSelected == true) {
        return true;
      } else {
        return false;
      }
    },

    getStartTime() {
      if (this.form.startDate == "") {
        if (this.form.startTime == "") {
          this.form.start = "";
        }
      } else {
        let d = "";
        if (this.form.startDate != "") {
          d = this.form.startDate;
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
        if (this.form.startTime != "") {
          t = this.form.startTime;
        } else {
          let today = new Date();
          t = today.getHours() + ":" + today.getMinutes();
        }
        let dt = new Date(d + " " + t);
        this.form.start = moment(dt)
          .unix();
      }
    },

    getEndTime() {
      if (this.form.endDate == "") {
        if (this.form.endTime == "") {
          this.form.end = "";
        }
      } else {
        let d = "";
        if (this.form.endDate != "") {
          d = this.form.endDate;
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
        if (this.form.endTime != "") {
          t = this.form.endTime;
        } else {
          let today = new Date();
          t = today.getHours() + ":" + today.getMinutes();
        }
        let dt = d + " " + t;
        this.form.end = moment(dt)
          .unix();
      }
    },

    getPayload() {
      let vlan = "";
      if(this.form.vlanVal != ""){
        vlan = this.form.vlanVal;
      }
      else{
        vlan = this.form.vlan;  
      }
      let payload = {
        name: this.form.name,
        duration: this.form.endDT,
        distro: this.form.eDistroName,
        profile: this.form.eProfileName,
        nodeCount: parseFloat(this.form.nodeCount),
        nodeList: this.form.nodes,
        start: this.form.startDT,
        vlan: vlan,
        group: this.form.grpname,
        description: this.form.description,
        noCycle: this.noCycle,
      };
      if (this.form.eDistroName === "") {
        this.$delete(payload, "distro");
      }
      if (this.form.eProfileName === "") {
        this.$delete(payload, "profile");
      }
      if (!this.form.startDT) {
        this.$delete(payload, "start");
      }
      if (!this.form.endDT) {
        this.$delete(payload, "duration");
      }
      if (this.form.nodeCount == 0) {
        if (this.form.nodes != "") {
          this.$delete(payload, "nodeCount");
        } else {
          this.validateFlag = false;
          alert("Either Node List or Node Count is required!");
        }
      } else if (this.form.nodeCount != 0) {
        if (this.form.nodes != "") {
          if(!this.selectedHosts){
            this.validateFlag = false;
            alert("You can ony enter either Node List or Node Count at a time!");
            this.form.nodeList = [];
            this.form.nodeListTextValue = "";
            this.form.nodeCount = 0;
          }
          else{
            this.$delete(payload, "nodeCount");  
          }
        }
      }
      if (this.form.nodes === "") {
        if (this.form.nodeCount != 0) {
          this.$delete(payload, "nodeList");
        }
      }
      if (vlan === "") {
        this.$delete(payload, "vlan");
      }
      if (this.form.description === "") {
        this.$delete(payload, "description");
      }
      if (this.form.grpname === "") {
        this.$delete(payload, "group");
      }
      if (!this.noCycle) {
        this.$delete(payload, "noCycle");
      }
      
      return payload;
    },

    getNodes() {
      if (this.nodeListText == true) {
        this.form.nodes = this.form.nodeList.toString();
      } else {
        this.form.nodes = this.form.nodeListTextValue;
      }
    },

    getStartDateOption() {
      if (this.dateTimeText == true) {
        this.form.startDT = this.form.start;
      } else {
        let dt = moment(String(this.form.dateTimeTextValue));
        this.form.startDT = moment(dt).unix();
      }
    },

    getEndDateOption() {
      if (this.endDateTimeText == true) {
        this.form.endDT = this.form.end;
      } else {
        let dt = moment(this.form.endDateTimeTextValue);
        this.form.endDT = moment(dt).unix();
      }
    },

    onSubmit(event) {
      event.preventDefault();
      if(this.hostsFromGrid.includes(",")) {
        this.form.nodeListTextValue = this.hostsFromGrid.split(", ").toString();
      } else {
        this.form.nodeListTextValue = this.hostsFromGrid.toString();  
      }
      
      this.form.nodeCount = this.nodeCountFromGrid;
      this.validateFlag = true;
      this.getStartTime();
      this.getStartDateOption();
      this.getEndTime();
      this.getEndDateOption();
      this.getNodes();
      let payload = this.getPayload();
      let createReservationUrl =
        this.$config.IGOR_API_BASE_URL + "/reservations";
      if (this.validateFlag) {
        axios
          .post(createReservationUrl, payload, { withCredentials: true })
          .then((response) => {
            payload = response.data.data.reservation[0];
            if (moment().unix() == payload.start) {
              this.updateHosts(payload);
            } else if (moment().unix() > payload.start) {
              this.updateHosts(payload);
            }

            this.$store.dispatch("insertNewReservations", payload);
            this.$store.dispatch("insertNewUserReservations", payload);
            this.updateProfile();
            alert("Reservation submitted sucessfully!");
          })
          .catch(function(error) {
            alert("Error: " + error.response.data.message);
          });
        this.onReset(event);
      }
    },

    updateHosts(payload) {
      let hosts = payload.hosts;
      let installError = payload.installError;
      hosts.forEach((element) => {
        if (this.$store.getters.hostsAvlPow.includes(element)) {
          let i = this.$store.getters.hostsAvlPow.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          if (installError != "") {
            this.$store.dispatch("updateHostsInstErrPow", payload);
          } else {
            this.$store.dispatch("removeHostsAvlPow", payload);
          }
        } else if (this.$store.getters.hostsAvlDown.includes(element)) {
          let i = this.$store.getters.hostsAvlDown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          if (installError != "") {
            this.$store.dispatch("updateHostsInstErrDown", payload);
          } else {
            this.$store.dispatch("removeHostsAvlDown", payload);
          }
        } else if (this.$store.getters.hostsAvlUnknown.includes(element)) {
          let i = this.$store.getters.hostsAvlUnknown.findIndex(
            (host) => host === element
          );
          let payload = {
            key: i,
            value: element,
          };
          if (installError != "") {
            this.$store.dispatch("updateHostsInstErrUnknown", payload);
          } else {
            this.$store.dispatch("removeHostsAvlUnknown", payload);
          }
        }
        let indexToRemove = this.$store.getters.hostsForResv.findIndex(
          (host) => host === element
        );
        this.$store.dispatch("removeHostsForResv", indexToRemove);
      });
    },

    updateProfile() {
      let profileUrl = this.$config.IGOR_API_BASE_URL + "/profiles";
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

    doNotCycle() {
      if (this.powerCycleStatus) {
        this.noCycle = true;
      }
    },

    onReset(event) {
      event.preventDefault();
      // Reset our form values
      this.form.description = "";
      this.form.nodeList = [];
      this.form.nodeCount = 0;
      this.form.vlan = "";
      this.form.name = "";
      this.form.grpname = "";
      this.form.eDistroName = "";
      this.form.eProfileName = "";
      this.form.eDistroProfileName = "";
      this.form.startTime = "";
      this.form.startDate = "";
      this.form.start = "";
      this.form.startDT = "";
      this.form.endTime = "";
      this.form.endDate = "";
      this.form.end = "";
      this.form.endDT = "";
      this.imageSelected = false;
      this.form.reservation = "";
      this.nodeListChecked = false;
      this.hostSelect = false;
      this.hostAllowSelect = false;
      this.nodeListText = false;
      this.form.nodeListTextValue = "";
      this.dateTimeText = false;
      this.form.dateTimeTextValue = "";
      this.dateTimeChecked = false;
      this.endDateTimeText = false;
      this.form.endDateTimeTextValue = "";
      this.endDateTimeChecked = false;
      this.powerCycleStatus = false;
      this.eDistroProfileNames = [];
      this.noCycle = false;
      this.vlanValue = false;
      this.form.vlanVal = "";
      // Trick to reset/clear native browser form validation state
      this.show = false;
      this.$store.dispatch('selectedResvHostID', []);
      this.$store.dispatch('selectedResvHosts', []);
      this.$store.dispatch('selectedResvHostsCount', 0);
      this.$nextTick(() => {
        this.show = true;
      });
    },
    
    next() {
      switch (this.form.reservation) {
        case "eDistro":
          this.eDistro = true;
          this.eProfile = false;
          this.form.eProfileName = "";
          this.imageSelected = true;
          this.eDistroProfileNames = this.eDistroNames;
          break;
        case "eProfile":
          this.eDistro = false;
          this.eProfile = true;
          this.form.eDistroName = "";
          this.imageSelected = true;
          this.eDistroProfileNames = this.eProfileNames;
          break;
        default:
          this.eDistro = false;
          this.eProfile = false;
          this.form.eDistroName = "";
          this.form.eProfileName = "";
          this.imageSelected = false;
          break;
      }
    },

    selectedProfileDistro(){
      if(this.form.reservation == "eDistro"){
        this.form.eDistroName = this.form.eDistroProfileName;
      }
      else if(this.form.reservation == "eProfile"){
        this.form.eProfileName = this.form.eDistroProfileName;  
      }
    },

    minmaxVal(){
      return this.vlanMin + " - " + this.vlanMax + " range"
    },
  },
};
</script>
