<template>
  <div class="top">
    <b-container fluid="true" class="row m-3">
      <b-row>
        <b-col>
          <b-button
            :aria-expanded="sidebarVisible"
            aria-controls="sidebar-1"
            class="btn bg-transparent btn-outline-light buttonfocus"
            @click="sidebarVisible = true"
          >
            <b-icon
              icon="text-start"
              aria-hidden="true"
              variant="dark"
              scale="1.5"
            ></b-icon>
          </b-button>
        </b-col>
      </b-row>
    </b-container>

    <b-offcanvas
      id="sidebar-1"
      v-model="sidebarVisible"
      placement="start"
      shadow
      backdrop
      title=""
    >
      <div class="px-2 py-2">
        <div class="sidebar p-1">
          <a class="active" href="#home" @click="sidebarVisible = false">Home</a>
          <a href="#contact" @click="sidebarVisible = false">Contact</a>
          <a href="#about" @click="sidebarVisible = false">About</a>
        </div>
      </div>
    </b-offcanvas>
  </div>
</template>

<script>
export default {
  data() {
    return {
      sidebarVisible: false,
    };
  },
  name: "SideMenu",
};
</script>
