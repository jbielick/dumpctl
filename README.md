## `dumpctl`

```hcl
database "myapp_production" {
  table "users" {
    where = "confirmed = 1"
    rule "mask" {
      columns = [table.ssn]
    }
  }
}
```
