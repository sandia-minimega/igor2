<template>
  <b-container class="node-grid" fluid>
    <b-row>
      <b-col>
        <b-button
          v-b-toggle.nodeLegend
          class="text-start buttonfocus fw-bold text-uppercase"
          variant="outline-transparent"
        >
          <b-icon icon="chevron-expand" class="me-2"></b-icon>
          {{ clusterName }}
        </b-button>
      </b-col>
      <b-col>
        <p class="text-end">
          {{ serverTime }}
        </p>
      </b-col>
    </b-row>
    <b-row class="mt-2">
      <b-col>
        <b-collapse id="nodeLegend" class="mt-2 w-100">
          <div class="legend">
            <div class="legend-section">
              <div class="legend-title fw-bold">
                RESERVATION / AVAILABILITY (background)
              </div>
              <b-row class="mt-2">
                <b-col
                    v-for="item in legendReservation"
                    :key="item.key"
                    cols="12"
                    md="6"
                    lg="4"
                    class="mb-2"
                >
                  <div class="d-flex align-items-start">
                    <div :class="['legend-swatch', item.swatchClass]">
                      {{ item.swatchText }}
                    </div>
                    <div class="ms-2">
                      <div class="fw-bold">{{ item.label }}</div>
                      <div class="legend-desc">{{ item.desc }}</div>
                    </div>
                  </div>
                </b-col>
              </b-row>
            </div>

            <div class="legend-section mt-3">
              <div class="legend-title fw-bold">
                POWER + CONNECTIVITY (foreground)
              </div>
              <b-row class="mt-2">
                <b-col
                    v-for="item in legendReadiness"
                    :key="item.key"
                    cols="12"
                    md="6"
                    lg="4"
                    class="mb-2"
                >
                  <div class="d-flex align-items-start">
                    <div class="legend-swatch card-item" :style="{ color: item.color }">
                      <b-icon :icon="item.icon" aria-hidden="true"></b-icon>
                    </div>
                    <div class="ms-2">
                      <div class="fw-bold">{{ item.label }}</div>
                      <div class="legend-desc">{{ item.desc }}</div>
                    </div>
                  </div>
                </b-col>
              </b-row>
              <div class="legend-note mt-2">
                Install error nodes will never appear as <span class="fw-bold">Up</span>.
              </div>
            </div>

            <div class="legend-section mt-3">
              <div class="legend-title text-uppercase fw-bold">
                Examples
              </div>
              <b-row class="mt-2">
                <b-col
                    v-for="ex in legendExamples"
                    :key="ex.key"
                    cols="12"
                    md="6"
                    lg="4"
                    class="mb-2"
                >
                  <div class="d-flex align-items-start">
                    <div :class="['legend-swatch', ex.className]">
                      <b-icon :icon="ex.icon" aria-hidden="true"></b-icon>
                    </div>
                    <div class="ms-2">
                      <div class="fw-bold">{{ ex.label }}</div>
                      <div class="legend-desc">{{ ex.desc }}</div>
                    </div>
                  </div>
                </b-col>
              </b-row>
            </div>
          </div>
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
            <span class="node-name">{{ host }}</span>
            <!-- <div class="tooltip-wrap">
              <div class="tooltip-content p-3 mb-2 bg-gradient-light">
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
      // Legend: background conveys reservation/access; text/icon color conveys readiness.
      legendReservation: [
        {
          key: 'available',
          swatchClass: 'card-item',
          swatchText: 'Available',
          label: 'Available',
          desc: 'Node can be reserved by anyone.',
        },
        {
          key: 'blocked',
          swatchClass: 'card-blocked-up',
          swatchText: 'Blocked',
          label: 'Blocked',
          desc: 'Node cannot be reserved. Current reservation persists until expiry. No ETA for return.',
        },
        {
          key: 'restricted',
          swatchClass: 'card-restricted-up',
          swatchText: 'Restr.',
          label: 'Restricted',
          desc: 'Node can only be reserved by certain users/groups or during a set time window. Only shown if you are not eligible.',
        },
        {
          key: 'reserved',
          swatchClass: 'card-reserved-up',
          swatchText: 'Reserved',
          label: 'Reserved',
          desc: 'Node is currently reserved by you.',
        },
        {
          key: 'group-reserved',
          swatchClass: 'card-grp-reserved-up',
          swatchText: 'Group',
          label: 'Group reserved',
          desc: 'Node is reserved and you are part of a group that can access the reservation.',
        },
        {
          key: 'other-reserved',
          swatchClass: 'card-other-reserved-up',
          swatchText: 'Other',
          label: 'Other reserved',
          desc: 'Node is currently reserved by another user or group.',
        },
        {
          key: 'insterr',
          swatchClass: 'card-insterr-displayonly',
          swatchText: 'Inst Err',
          label: 'Install error',
          desc: 'Startup failed during an active reservation. Will not show readiness as Up/Ready.',
        },
      ],

      legendReadiness: [
        {
          key: 'off',
          icon: 'circle-fill',
          color: 'red',
          label: 'Off',
          desc: 'Node has no power. Normal for unused nodes; attention-worthy if reserved.',
        },
        {
          key: 'on',
          icon: 'circle-fill',
          color: 'cornflowerblue',
          label: 'On',
          desc: 'Node has power but no network connectivity.',
        },
        {
          key: 'unknown',
          icon: 'question-circle-fill',
          color: 'dimgray',
          label: 'Unknown',
          desc: 'Power status could not be obtained.',
        },
        {
          key: 'up',
          icon: 'circle-fill',
          color: 'white',
          label: 'Up / Ready',
          desc: 'Node responds to TCP requests (login-ready).',
        },
        {
          key: 'ping',
          icon: 'circle-fill',
          color: 'gold',
          label: 'Ping',
          desc: 'Node responds only to ICMP ping (not yet login-ready).',
        },
      ],

      legendExamples: [
        {
          key: 'ex-reserved-up',
          className: 'card-reserved-up',
          icon: 'circle-fill',
          label: 'Reserved + Up',
          desc: 'Reserved by you and login-ready.',
        },
        {
          key: 'ex-reserved-off',
          className: 'card-reserved-off',
          icon: 'circle-fill',
          label: 'Reserved + Off',
          desc: 'Reserved, but currently powered down.',
        },
        {
          key: 'ex-insterr-ping',
          className: 'card-insterr-ping',
          icon: 'circle-fill',
          label: 'Install error + Ping',
          desc: 'Reservation startup failed; node responds to ping.',
        },
      ],
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
      // create an array with all numbers between clickedNodeID and lastClickedNodeID
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
      // create an array with all numbers between clickedNodeID and lastClickedNodeID
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

      // 1) Highest priority: blocked (maintenance)
      if (this.hostsBlockedPow.includes(host)) return "card-item card-blocked card-up";
      if (this.hostsBlockedPing.includes(host)) return "card-item card-blocked card-ping";
      if (this.hostsBlockedOn.includes(host)) return "card-item card-blocked card-on";
      if (this.hostsBlockedDown.includes(host)) return "card-item card-blocked card-off";
      if (this.hostsBlockedUnknown.includes(host)) return "card-item card-blocked card-unknown";

      // 2) Next: install error
      if (this.hostsInstErrPow.includes(host)) return "card-item card-insterr card-up";
      if (this.hostsInstErrPing.includes(host)) return "card-item card-insterr card-ping";
      if (this.hostsInstErrOn.includes(host)) return "card-item card-insterr card-on";
      if (this.hostsInstErrDown.includes(host)) return "card-item card-insterr card-off";
      if (this.hostsInstErrUnknown.includes(host)) return "card-item card-insterr card-unknown";

      // 3) Next: restricted
      if (this.hostsRestrictedPow.includes(host)) return "card-item card-restricted card-up";
      if (this.hostsRestrictedPing.includes(host)) return "card-item card-restricted card-ping";
      if (this.hostsRestrictedOn.includes(host)) return "card-item card-restricted card-on";
      if (this.hostsRestrictedDown.includes(host)) return "card-item card-restricted card-off";
      if (this.hostsRestrictedUnknown.includes(host)) return "card-item card-restricted card-unknown";

      // 4) Reserved buckets (your user / group / other)
      if (this.hostsResvUp.includes(host)) return "card-item card-reserved card-up";
      if (this.hostsResvPing.includes(host)) return "card-item card-reserved card-ping";
      if (this.hostsResvOn.includes(host)) return "card-item card-reserved card-on";
      if (this.hostsResvDown.includes(host)) return "card-item card-reserved card-off";
      if (this.hostsResvUnknown.includes(host)) return "card-item card-reserved card-unknown";

      if (this.hostsGrpResvPow.includes(host)) return "card-item card-grp-reserved card-up";
      if (this.hostsGrpResvPing.includes(host)) return "card-item card-grp-reserved card-ping";
      if (this.hostsGrpResvOn.includes(host)) return "card-item card-grp-reserved card-on";
      if (this.hostsGrpResvDown.includes(host)) return "card-item card-grp-reserved card-off";
      if (this.hostsGrpResvUnknown.includes(host)) return "card-item card-grp-reserved card-unknown";

      if (this.hostsOtherResvPow.includes(host)) return "card-item card-other-reserved card-up";
      if (this.hostsOtherResvPing.includes(host)) return "card-item card-other-reserved card-ping";
      if (this.hostsOtherResvOn.includes(host)) return "card-item card-other-reserved card-on";
      if (this.hostsOtherResvDown.includes(host)) return "card-item card-other-reserved card-off";
      if (this.hostsOtherResvUnknown.includes(host)) return "card-item card-other-reserved card-unknown";


      if (this.hostsAvlDown.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-item card-selected card-off"
        } else {
          return "card-item card-available card-off";
        }
      } else if (this.hostsAvlUnknown.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-item card-selected card-unknown"
        } else {
        return "card-item card-available card-unknown"
        }
      } else if (this.hostsAvlPing.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-item card-selected card-ping"
        } else {
          return "card-item"
        }
      } else if (this.hostsAvlOn.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-item card-selected card-on"
        } else {
          return "card-item"
        }
      } else if (this.hostsAvlPow.includes(host)) {
        if(this.selectedHosts.includes(host)) {
          return "card-item card-selected card-up"
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
    hostsResvUp() {
      return this.$store.getters.hostsResvUp;
    },
    hostsResvOn() {
      return this.$store.getters.hostsResvOn;
    },
    hostsResvPing() {
      return this.$store.getters.hostsResvPing;
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
    hostsGrpResvOn() {
      return this.$store.getters.hostsGrpResvOn;
    },
    hostsGrpResvPing() {
      return this.$store.getters.hostsGrpResvPing;
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
    hostsOtherResvOn() {
      return this.$store.getters.hostsOtherResvOn;
    },
    hostsOtherResvPing() {
      return this.$store.getters.hostsOtherResvPing;
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
    hostsAvlPing() {
      return this.$store.getters.hostsAvlPing;
    },
    hostsAvlOn() {
      return this.$store.getters.hostsAvlOn;
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
    hostsBlockedOn() {
      return this.$store.getters.hostsBlockedOn;
    },
    hostsBlockedPing() {
      return this.$store.getters.hostsBlockedPing;
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
    hostsInstErrOn() {
      return this.$store.getters.hostsInstErrOn;
    },
    hostsInstErrPing() {
      return this.$store.getters.hostsInstErrPing;
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
    hostsRestrictedOn() {
      return this.$store.getters.hostsRestrictedOn;
    },
    hostsRestrictedPing() {
      return this.$store.getters.hostsRestrictedPing;
    },
    hostsRestrictedPow() {
      return this.$store.getters.hostsRestrictedOn;
    },
    selectedHosts(){
      return this.$store.getters.selectedHosts;
    },
    hostSelectedPow(){
      return this.$store.getters.hostSelectedPow;
    },
    hostSelectedPing(){
      return this.$store.getters.hostSelectedPing;
    },
    hostSelectedOn(){
      return this.$store.getters.hostSelectedOn;
    },
    hostSelectedDown(){
      return this.$store.getters.hostSelectedDown;
    },
    hostSelectedUnknown(){
      return this.$store.getters.hostSelectedUnknown;
    },
    gridStyle() {
      return {
        gridTemplateColumns: `repeat(auto-fit, minmax(7ch, 1fr))`,
        gridAutoRows: `40px`, // <-- makes every cell taller
      };
    },
  },
};
</script>

<style scoped>
.legend {
  border-radius: 4px;
}

.legend-title {
  font-size: 0.85rem;
}

.legend-desc {
  font-size: 0.8rem;
  color: #6c757d; /* Bootstrap text-muted */
  line-height: 1.2;
}

.legend-note {
  font-size: 0.8rem;
  color: #6c757d;
}

.legend-swatch {
  min-width: 78px;
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  padding-left: 6px;
  padding-right: 6px;
  font-weight: bold;
  user-select: none;
  pointer-events: none;
}

html.theme-dark .legend-title,
html.theme-dark .legend-section .fw-bold {
  color: #f1f5fb;
}

html.theme-dark .legend-desc,
html.theme-dark .legend-note {
  color: #c1cad8;
}
</style>
