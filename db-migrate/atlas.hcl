// atlas must be installed on dev machine to run this
// follows atlas instructions online to download dependencies needed
// uncomment the import statement in tools.go
// in db-migrate folder run command:  atlas migrate diff --env gorm
// run to fix hash mismatch:          atlas migrate hash

data "external_schema" "gorm" {
  program = [
    "go",
    "run",
    "-mod=mod",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "../internal/app/igor-server",
    "--dialect", "sqlite", // | postgres | sqlite | sqlserver
  ]
}

env "gorm" {
  // this points to the definition above and reads changes to your db tables/GORM definitions
  src = data.external_schema.gorm.url
  dev = "sqlite://dev?mode=memory"
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}