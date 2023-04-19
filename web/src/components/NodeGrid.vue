<template>
  <b-container class="node-grid" fluid>
    <b-row>
      <b-col>
        <b-button
          v-b-toggle.nodeLegend
          class="text-left buttonfocus font-weight-bold text-uppercase"
          variant="outline-transparent"
        >
          <b-icon icon="chevron-expand" class="mr-2"></b-icon>
          {{ clusterName }}
        </b-button>
      </b-col>
      <b-col>
        <p class="text-right">
          {{ serverTime }}
        </p>
      </b-col>
    </b-row>
    <b-row class="mt-2">
      <b-col>
        <b-collapse id="nodeLegend" class="mt-2 w-100">
          <table class="w-100">
            <tr>
              <td colspan="3">
                <div class="card-item font-weight-bold">
                  Available
                </div>
              </td>
              <td colspan="3">
                <div class="card-reserved-powered font-weight-bold">
                  Reserved
                </div>
              </td>
              <td colspan="3">
                <div class="card-grp-reserved-powered font-weight-bold">
                  Group Resv
                </div>
              </td>
              <td colspan="3">
                <div class="card-other-reserved-powered font-weight-bold">
                  Other Resv
                </div>
              </td>
              <td colspan="3">
                <div class="card-blocked-powered font-weight-bold">
                  Blocked
                </div>
              </td>
              <td colspan="3">
                <div class="card-restricted-powered font-weight-bold">
                  Restricted
                </div>
              </td>
              <td colspan="3">
                <div class="card-insterr-powered font-weight-bold">
                  Inst Err
                </div>
              </td>
            </tr>
            <tr>
              <td>
                <div class="card-item font-weight-bold">
                  <b-icon-arrow-up-circle-fill></b-icon-arrow-up-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-available-off  font-weight-bold">
                  <b-icon-arrow-down-circle-fill></b-icon-arrow-down-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-available-unknown  font-weight-bold">
                  <b-icon-question-circle-fill></b-icon-question-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-reserved-powered font-weight-bold">
                  <b-icon-arrow-up-circle-fill></b-icon-arrow-up-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-reserved-off font-weight-bold">
                  <b-icon-arrow-down-circle-fill></b-icon-arrow-down-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-reserved-unknown font-weight-bold">
                  <b-icon-question-circle-fill></b-icon-question-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-grp-reserved-powered font-weight-bold">
                  <b-icon-arrow-up-circle-fill></b-icon-arrow-up-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-grp-reserved-off font-weight-bold">
                  <b-icon-arrow-down-circle-fill></b-icon-arrow-down-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-grp-reserved-unknown font-weight-bold">
                  <b-icon-question-circle-fill></b-icon-question-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-other-reserved-powered font-weight-bold">
                  <b-icon-arrow-up-circle-fill></b-icon-arrow-up-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-other-reserved-off  font-weight-bold">
                  <b-icon-arrow-down-circle-fill></b-icon-arrow-down-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-other-reserved-unknown  font-weight-bold">
                  <b-icon-question-circle-fill></b-icon-question-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-blocked-powered font-weight-bold">
                  <b-icon-arrow-up-circle-fill></b-icon-arrow-up-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-blocked-off  font-weight-bold">
                  <b-icon-arrow-down-circle-fill></b-icon-arrow-down-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-blocked-unknown  font-weight-bold">
                  <b-icon-question-circle-fill></b-icon-question-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-restricted-powered font-weight-bold">
                  <b-icon-arrow-up-circle-fill></b-icon-arrow-up-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-restricted-off  font-weight-bold">
                  <b-icon-arrow-down-circle-fill></b-icon-arrow-down-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-restricted-unknown  font-weight-bold">
                  <b-icon-question-circle-fill></b-icon-question-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-insterr-powered font-weight-bold">
                  <b-icon-arrow-up-circle-fill></b-icon-arrow-up-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-insterr-off  font-weight-bold">
                  <b-icon-arrow-down-circle-fill></b-icon-arrow-down-circle-fill>
                </div>
              </td>
              <td>
                <div class="card-insterr-unknown  font-weight-bold">
                  <b-icon-question-circle-fill></b-icon-question-circle-fill>
                </div>
              </td>
            </tr>
          </table>
        </b-collapse>
      </b-col>
    </b-row>

    <b-row class="mt-2">
      <b-col>
        <b-list-group :style="gridStyle" class="card-list grid" id="hostItem" multiselect>
          <b-list-group-item
            button
            v-for="(host, index) in this.hostNames"
            :key="index"
            v-bind:class="hostStatus(host)"
            v-on:click.shift="hostClick(index)"   
            v-on:click="nodeClickedListener(index)" 
            v-on:click.ctrl="nodeCtrlClickedListener(index)"
            v-on:mouseover.shift="onHover(index)"
            @mousedown="startNode(index)"   
            @mouseup="afterSelect(index)"
          >
            {{ host }}
            <!-- <div class="tooltip-wrap">
              <div class="tooltip-content p-3 mb-2 bg-gradient-light text-dark">
                <p>
                  {{ host.cluster }}
                  {{ host.state }}
                </p>
              </div>
            </div> -->
          </b-list-group-item>
        </b-list-group>
      </b-col>
    </b-row>
  </b-container>
</template>

<script>
export default {
  name: "NodeGrid",
  data() {
    return {
      shiftStart: null,
      shiftEnd: null,
      lastClickedNode: null,
    };
  },
  methods: {
    afterSelect(index){
      this.shiftStart = null;
      this.shiftEnd = null;
    },

    startNode(index){
      this.shiftStart = index;
    },

    onHover(index){
      var selectedNodes = [];
      // create an array with all numbers betwen clickedNodeID and lastClickedNodeID
      let minNodeID = Math.min(index, this.lastClickedNode);
      let maxNodeID = Math.max(index, this.lastClickedNode);
      let nodesInRange = Array.from(Array(maxNodeID - minNodeID + 1), (_, i) => i + minNodeID);
      if (selectedNodes.includes(index)) {
        // unselect the nodes in range if the node being clicked is already selected
        selectedNodes = selectedNodes.filter(nodeID => !nodesInRange.includes(nodeID));
      } else {
        // select the nodes in range if the node being clicked is not already selected
        selectedNodes = selectedNodes.concat(nodesInRange);
      }
      let nodeNames = [];
      selectedNodes.forEach(element => {
        nodeNames.push(this.hostNames[element]);
      });
      this.$store.dispatch('selectedResvHosts', nodeNames);
    },

    nodeClickedListener(clickedNode) {
      this.lastClickedNode = clickedNode;
    },

    hostClick(clickedNode) {
      var selectedNodes = [];
      // create an array with all numbers betwen clickedNodeID and lastClickedNodeID
      let minNodeID = Math.min(clickedNode, this.lastClickedNode);
      let maxNodeID = Math.max(clickedNode, this.lastClickedNode);
      let nodesInRange = Array.from(Array(maxNodeID - minNodeID + 1), (_, i) => i + minNodeID);
      if (selectedNodes.includes(clickedNode)) {
        // unselect the nodes in range if the node being clicked is already selected
        selectedNodes = selectedNodes.filter(nodeID => !nodesInRange.includes(nodeID));
      } else {
        // select the nodes in range if the node being clicked is not already selected
        selectedNodes = selectedNodes.concat(nodesInRange);
      }
      let nodeNames = [];
      selectedNodes.forEach(element => {
        nodeNames.push(this.hostNames[element]);
      });
      this.$store.dispatch('selectedResvHosts', nodeNames);      
    },

    nodeCtrlClickedListener(clickedNode) {
      var selectedNodes = this.$store.getters.selectedHostID;

      // check if node is already selected
      if (selectedNodes.includes(clickedNode)) {
        selectedNodes = selectedNodes.filter(val => clickedNode!= val);
      } else {
        selectedNodes.push(clickedNode);
      }
      this.$store.dispatch('selectedResvHostID', selectedNodes);
      
      let nodeNames = [];
      selectedNodes.forEach(element => {
        nodeNames.push(this.hostNames[element]);
      });
      this.$store.dispatch('selectedResvHosts', nodeNames);
      console.log(this.$store.getters.selectedHosts);
    },
    
    hostStatus(host) {
      if (this.hostsResvPow.includes(host)) {
        return "card-reserved-powered";
      } else if (this.hostsResvDown.includes(host)) {
        return "card-reserved-off";
      } else if (this.hostsResvUnknown.includes(host)) {
        return "card-reserved-unknown";
      } else if (this.hostsGrpResvPow.includes(host)) {
        return "card-grp-reserved-powered";
      } else if (this.hostsGrpResvDown.includes(host)) {
        return "card-grp-reserved-off";
      } else if (this.hostsGrpResvUnknown.includes(host)) {
        return "card-grp-reserved-unknown";
      } else if (this.hostsOtherResvUnknown.includes(host)) {
        return "card-other-reserved-unknown";
      } else if (this.hostsOtherResvPow.includes(host)) {
        return "card-other-reserved-powered";
      } else if (this.hostsOtherResvDown.includes(host)) {
        return "card-other-reserved-off";
      } else if (this.hostsAvlDown.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-selected-off"
        } else {
          return "card-available-off";
        }
      } else if (this.hostsAvlUnknown.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-selected-unknown"
        } else {
        return "card-available-unknown"
        }
      } else if (this.hostsBlockedUnknown.includes(host)) {
        return "card-blocked-unknown";
      } else if (this.hostsBlockedPow.includes(host)) {
        return "card-blocked-powered";
      } else if (this.hostsBlockedDown.includes(host)) {
        return "card-blocked-off";
      } else if (this.hostsInstErrUnknown.includes(host)) {
        return "card-insterr-unknown";
      } else if (this.hostsInstErrPow.includes(host)) {
        return "card-insterr-powered";
      } else if (this.hostsInstErrDown.includes(host)) {
        return "card-insterr-off";
      } else if (this.hostsRestrictedUnknown.includes(host)) {
        return "card-restricted-unknown";
      } else if (this.hostsRestrictedPow.includes(host)) {
        return "card-restricted-powered";
      } else if (this.hostsRestrictedDown.includes(host)) {
        return "card-restricted-off";
      } else if (this.hostsAvlPow.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-selected-powered"
        } else {
          return "card-item";
        }
      } 
    },
  },
  computed: {
    serverTime() {
      return this.$store.getters.serverTime;
    },
    hostNames() {
      return this.$store.getters.hostNames;
    },
    reservations() {
      return this.$store.getters.reservations;
    },
    clusterName() {
      return this.$store.state.clusterName;
    },
    hostsOtherReserved() {
      return this.$store.getters.hostsOtherReserved;
    },
    hostsResvPow() {
      return this.$store.getters.hostsResvPow;
    },
    hostsResvDown() {
      return this.$store.getters.hostsResvDown;
    },
    hostsResvUnknown() {
      return this.$store.getters.hostsResvUnknown;
    },
    hostsGrpResvPow() {
      return this.$store.getters.hostsGrpResvPow;
    },
    hostsGrpResvDown() {
      return this.$store.getters.hostsGrpResvDown;
    },
    hostsGrpResvUnknown() {
      return this.$store.getters.hostsGrpResvUnknown;
    },
    hostsOtherResvPow() {
      return this.$store.getters.hostsOtherResvPow;
    },
    hostsOtherResvDown() {
      return this.$store.getters.hostsOtherResvDown;
    },
    hostsOtherResvUnknown() {
      return this.$store.getters.hostsOtherResvUnknown;
    },
    hostsAvlPow() {
      return this.$store.getters.hostsAvlPow;
    },
    hostsAvlDown() {
      return this.$store.getters.hostsAvlDown;
    },
    hostsAvlUnknown() {
      return this.$store.getters.hostsAvlUnknown;
    },
    hostsBlockedUnknown() {
      return this.$store.getters.hostsBlockedUnknown;
    },
    hostsBlockedDown() {
      return this.$store.getters.hostsBlockedDown;
    },
    hostsBlockedPow() {
      return this.$store.getters.hostsBlockedPow;
    },
    hostsInstErrUnknown() {
      return this.$store.getters.hostsInstErrUnknown;
    },
    hostsInstErrDown() {
      return this.$store.getters.hostsInstErrDown;
    },
    hostsInstErrPow() {
      return this.$store.getters.hostsInstErrPow;
    },
    hostsRestrictedUnknown() {
      return this.$store.getters.hostsRestrictedUnknown;
    },
    hostsRestrictedDown() {
      return this.$store.getters.hostsRestrictedDown;
    },
    hostsRestrictedPow() {
      return this.$store.getters.hostsRestrictedPow;
    },
    selectedHosts(){
      return this.$store.getters.selectedHosts;
    },
    hostSelectedPow(){
      return this.$store.getters.hostSelectedPow;
    },
    hostSelectedDown(){
      return this.$store.getters.hostSelectedDown;
    },
    hostSelectedUnknown(){
      return this.$store.getters.hostSelectedUnknown;
    },
    gridStyle() {
      return {
        gridTemplateColumns: `repeat(auto-fit, minmax(50px, 1fr))`,
      };
    },
  },
};
</script>
