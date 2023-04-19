const path = require("path");
module.exports = {
  outputDir: path.resolve(__dirname, "./dist"),
  devServer: {
    // NOTE: The setting below forces Vue to use its own embedded self-signed cert. We can
    // specify the key and cert file if we want to test the dev server using real ones.
    //
    // https: {
    //     key: fs.readFileSync('path to key file'),
    //     cert: fs.readFileSync('path to cert file'),
    // },
    https: true,
    public: "https://localhost:3000/",
  },
};
