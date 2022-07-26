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


Roadmap:

- enhance sampling with CTE and window function where supported (mysql >=8)
- support more dialects
