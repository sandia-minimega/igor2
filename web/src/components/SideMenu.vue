<template>
  <div class="top">
    <b-container fluid="true" class="row m-3">
      <b-row>
        <b-col>
          <b-button
            v-b-toggle.sidebar-1
            class="btn bg-transparent btn-outline-light text-dark buttonfocus"
          >
            <b-icon
              icon="text-left"
              aria-hidden="true"
              variant="dark"
              scale="1.5"
            ></b-icon>
          </b-button>
        </b-col>
      </b-row>
    </b-container>

    <b-sidebar id="sidebar-1" shadow v-b-tooltip.hover title="">
      <div class="px-2 py-2">
        <div class="sidebar p-1">
          <a class="active" href="#home">Home</a>
          <a href="#contact">Contact</a>
          <a href="#about">About</a>
        </div>
      </div>
    </b-sidebar>
  </div>
</template>

<script>
export default {
  data() {
    return {};
  },
  name: "SideMenu",
};
</script>
