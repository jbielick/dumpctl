database "myapp_production" {
  destination_database = "myapp_test"
  table "users" {
    where = "setup_complete = 1"
    rule "mask" {
      columns = [pin]
    }
  }
  table "appointments" {
    where = "created_at > curdate() - interval 7 day"
    where {
      user_id = users.id
    }
  }
  table "comments" {
    where {
      appointment_id = appointments.id
    }
  }
}
