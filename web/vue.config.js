const path = require("path");
const Components = require("unplugin-vue-components/webpack");
const {
  BootstrapVueNextResolver,
} = require("bootstrap-vue-next/resolvers");

module.exports = {
  css: {
    loaderOptions: {
      scss: {
        sassOptions: {
          quietDeps: true,
        },
      },
    },
  },
  outputDir: path.resolve(__dirname, "./dist"),
  configureWebpack: {
    plugins: [
      Components({
        dts: false,
        resolvers: [BootstrapVueNextResolver()],
      }),
    ],
  },
  devServer: {
    // NOTE: The setting below forces Vue to use its own embedded self-signed cert. We can
    // specify the key and cert file if we want to test the dev server using real ones.
    //
    // https: {
    //     key: fs.readFileSync('path to key file'),
    //     cert: fs.readFileSync('path to cert file'),
    // },
    //https: true,
    //public: "https://localhost:3000/",
    server: 'https',
    port: 3000,
    webSocketServer: 'ws',
    client: {
      webSocketTransport: 'ws',
      webSocketURL: 'ws://0.0.0.0:3000/ws',
    },
  },
};
