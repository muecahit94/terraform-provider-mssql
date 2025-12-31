data "mssql_query" "example" {
  database_name = "master"
  query         = "SELECT @@VERSION as version"
}

output "server_version" {
  value = data.mssql_query.example.result[0].values.version
}
