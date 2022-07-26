database "myapp_production" {
  destination_database = "myapp_test"
  table "users" {
    where = "pin is not null"
    rule "mask" {
      columns = [pin]
    }
  }
}
