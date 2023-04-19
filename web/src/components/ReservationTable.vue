<template>
  <div class="mt-3">
    <b-card no-body>
      <b-tabs card active-nav-item-class="font-weight-bold">
        <b-tab active no-body title="Reservations">
          <b-row class="mt-3">
            <b-col lg="6">
              <b-form-group
                label=""
                label-for="filter-input"
                label-cols-sm="3"
                label-align-sm="right"
                label-size="sm"
                class="mb-0"
              >
                <b-input-group size="sm">
                  <b-form-input
                    id="filter-input"
                    v-model="filter"
                    type="search"
                    placeholder="Search"
                  ></b-form-input>

                  <b-input-group-append>
                    <b-button :disabled="!filter" @click="filter = ''"
                      >Clear</b-button
                    >
                  </b-input-group-append>
                </b-input-group>
              </b-form-group>
            </b-col>
            <b-col sm="3">
              <b-form-group
                label="Show"
                label-for="per-page-select"
                label-cols-sm="6"
                label-cols-md="4"
                label-cols-lg="3"
                label-align-sm="right"
                label-size="sm"
                class="mb-0"
              >
                <b-form-select
                  id="per-page-select"
                  v-model="perPage"
                  :options="pageOptions"
                  size="sm"
                ></b-form-select>
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
                responsive="sm"
                :current-page="currentPage"
                :per-page="perPage"
                class="rtable pl-3 pr-3 pb-3"
                show-empty
                :filter="filter"
                :filter-included-fields="filterOn"
                @filtered="onFiltered"
              >
                <template #empty="scope">
                  <h6 class="font-italic">{{ scope.emptyText }}</h6>
                </template>
              </b-table>
            </b-col>
          </b-row>
          <b-row>
            <b-col>
              <b-pagination
                v-model="currentPage"
                :total-rows="totalRows"
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
import moment from "moment";
import SmartTable from "vuejs-smart-table";
import ref from "vue";
Vue.use(SmartTable);
export default {
  name: "ReservationTable",
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
          key: "owner",
          thClass: "theader",
          thStyle: "font-weight: bold",
        },
        {
          sortable: true,
          key: "start",
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
      ],
      search: null,
      column: null,
      currentSort: "pn",
      currentSortDir: "asc",
      currentPage: 1,
      perPage: 5,
      totalPages: 0,
      expiring: false,
      pageOptions: [5, 10, 15, 100],
      filter: null,
      filterOn: [],
    };
  },
  methods: {
    onFiltered(filteredItems) {
      // Trigger pagination to update the number of buttons/pages due to filtering
      this.totalRows = filteredItems.length;
    },
    sort: function(col) {
      // if you click the same label twice
      if (this.currentSort == col) {
        this.currentSortDir = this.currentSortDir === "asc" ? "desc" : "asc";
      } else {
        this.currentSort = col;
        console.log("diff col: " + col);
      }
    },
    colValue: function(colName, colValue) {
      if (colName === "end" || colName === "start") {
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
  },

  computed: {
    sortOptions() {
      // Create an options list from our fields
      return this.fields
        .filter((f) => f.sortable)
        .map((f) => {
          return { text: f.label, value: f.key };
        });
    },

    reservations() {
      return this.$store.getters.reservations;
    },
    totalRows: {
      get(){
        return this.$store.getters.reservationsFilteredLength;
      },
      set(newValue){
        this.$store.dispatch("insertReservationsForFiltering", newValue);  
      }
    },
    rows() {
      if (!this.reservations.length) {
        return [];
      }

      return this.reservations
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
};
</script>
