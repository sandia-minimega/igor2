import Vue from "vue";
import Router from "vue-router";
import Login from "./components/Login.vue";
import Secure from "./components/Secure.vue";
import NotFound from "./components/error-pages/NotFound.vue";
import NodeGrid from "./components/NodeGrid.vue";
import ReservationTable from "./components/ReservationTable.vue";
import UserView from "./components/UserView.vue";
import ProfileTable from "./components/ProfileTable.vue";
import DistroTable from "./components/DistroTable.vue";
import HomeTab from "./components/HomeTab.vue";
import CreateDistro from "./components/CreateDistro.vue";
import CreateReservation from "./components/CreateReservation.vue";
import TabMenu from "./components/TabMenu.vue";
import UserReservations from "./components/UserReservations.vue";
import SideMenu from "./components/SideMenu.vue";
import CreateGroup from "./components/CreateGroup.vue";
import CreateProfile from "./components/CreateProfile.vue";

Vue.use(Router);
let router = new Router({
  mode: "history",
  routes: [
    {
      path: "/",
      name: "login",
      component: Login,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/login",
      name: "login",
      component: Login,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/secure",
      name: "secure",
      component: Secure,
      meta: {
        requiresAuth: true,
        meta: {
          requiresAuth: false,
        },
      },
    },
    {
      path: "/nodegrid",
      name: "nodegrid",
      component: NodeGrid,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/reservationtable",
      name: "reservationtable",
      component: ReservationTable,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/userview",
      name: "userview",
      component: UserView,
      meta: {
        requiresAuth: true,
      },
    },

    {
      path: "/profiletable",
      name: "profiletable",
      component: ProfileTable,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/distrotable",
      name: "distrotable",
      component: DistroTable,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/hometab",
      name: "hometab",
      component: HomeTab,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/createdistro",
      name: "createdistro",
      component: CreateDistro,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/createreservation",
      name: "createreservation",
      component: CreateReservation,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/tabmenu",
      name: "tabmenu",
      component: TabMenu,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/sidemenu",
      name: "sidemenu",
      component: SideMenu,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/userreservations",
      name: "userreservations",
      component: UserReservations,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/creategroup",
      name: "creategroup",
      component: CreateGroup,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "/createprofile",
      name: "createprofile",
      component: CreateProfile,
      meta: {
        requiresAuth: false,
      },
    },
    {
      path: "*",
      name: "NotFound",
      component: NotFound,
      meta: {
        requiresAuth: false,
      },
    },
  ],
});
router.beforeEach((to, from, next) => {
  if (to.fullPath === '/userview') {
    if (sessionStorage.getItem("authenticated")) {
      next();
    }
    else {
      next("/login");
    } 
  }
  else{
    next();
  }
});
export default router;
