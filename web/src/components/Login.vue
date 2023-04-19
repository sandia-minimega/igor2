<template>
  <div class="form-group form-group-sm offset-md-3 p-3">
    <form class="login" @submit.prevent="login">
      <h1>Sign in</h1>
      <div>
        <label>Username</label>
        <input
          required
          v-model="username"
          type="text"
          placeholder="Name"
          class="form-control col-sm-3"
        />
      </div>
      <div>
        <label>Password</label>
        <input
          required
          v-model="password"
          type="password"
          placeholder="Password"
          class="form-control col-sm-3"
        />
      </div>
      <div>
        <hr />
        <button class="btn btn-primary" type="submit">Login</button>
      </div>
    </form>
  </div>
</template>

<script>
export default {
  name: "Login",
  data() {
    return {
      username: "",
      password: "",
    };
  },
  methods: {
    login: function() {
      let username = this.username;
      let password = this.password;
      this.$store
        .dispatch("login", { username, password })
        .then((response) => {
          this.$router.push("/userview");
        })
        .catch(function(error) {
          if (error.response.status === 401) {
            alert("Error: " + error.response.data.message);
            window.location.reload();
          }
        });
    },
  },
};
</script>
