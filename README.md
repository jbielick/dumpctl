## `dumpctl`

Take control of `mysqldump`.

`dumpctl` is a drop-in replacement for `mysqldump` that reads a declarative HCL configuration file instructing the process to:

1. Use a `where` clause to limit a table's dump
2. Define foreign keys to dump related records
3. Sample the table contents while dumping to avoid a full dump
4. **Obfuscate, de-identify, or redact column data from dumped tables**

The resulting output is equivalent to the mysqldump output of insert statements. This can be useful for test data setup.

Before:

```sql
INSERT INTO `users` VALUES ("Ryder Mckenzie",  7400);
INSERT INTO `users` VALUES ("Dennis Salas",    1189);
INSERT INTO `users` VALUES ("Toby Flenderson", null);
INSERT INTO `users` VALUES ("Brenda Padilla",  3672);
INSERT INTO `users` VALUES ("Yetta Bryant",    4470);
INSERT INTO `users` VALUES ("Demetria Benton", 2819);
```


```hcl
// config.hcl
database "myapp_production" {
  table "users" {
    where = "pin is not null"
    rule "mask" {
      columns = [pin]
    }
  }
}
```

`dumpctl -c config.hcl -u root -p root`

After:

```sql
INSERT INTO `users` VALUES (1,  'Ryder Mckenzie',   '****');
INSERT INTO `users` VALUES (3,  'Brenda Padilla',   '****');
INSERT INTO `users` VALUES (4,  'Yetta Bryant',     '****');
INSERT INTO `users` VALUES (5,  'Demetria Benton',  '****');
```

## Configuration

You can write a declarative configuration in [HCL](https://github.com/hashicorp/hcl).

### Database configuration

Top-level blocks in the configuration file are databases to be dumped. Declare a database to be dumped like so:

```hcl
database "myapp_production" {
  // ...
}
```

**By default, no tables are dumped. To include a table in the dump, declare it with a block (see below).**

A database block accepts one optional attribute, `destination_database`. If you'd like the dumped data to be inserted into a different database than where it was sourced, use this attribute to change the name.

```hcl
database "myapp_production" {
  destination_database = "myapp_test"
}
```

### Table configuration

A table can be included in the dump by declaring a block with the table name.

```hcl
database "myapp_production" {
  destination_database = "myapp_test"

  table "users" {}
}
```

The dump for this configuration will include only the `users` table with no modifications. To add more tables, simply add more blocks like the `table "users"` block.

#### Selecting data for dumping

The default behavior is to dump all data for a table. However, this can often produce more data than desired or practically usable for the end-use case. You can select a subset of the data to dump by using the `where` attribute like so:

```hcl
database "myapp_production" {
  destination_database = "myapp_test"

  table "users" {
    where = "setup_complete = 1"
  }
}
```

The dump for this configuration will include all of the records from the `users` table where the `setup_complete` column is equal to `1` (truthy).

#### Sampling data

When filtering records with `where` is not sufficient, records from a table can be sampled for dumping by providing a `sample_rate` attribute for the table and setting it to a decimal representing the percentage of records desired.

```hcl
database "myapp_production" {
  destination_database = "myapp_test"

  table "users" {
    sample_rate = 0.05
  }
}
```

The dump for this configuration will include 5% of the records from the `users` table.

#### Related records

Sampling or filtering records is not very useful if the related data cannot also be reduced, so related tables can provide "join conditions" (in quotes because its not actually a `join`) to dump only the data related to the subset of data dumped from prior table.

```hcl
database "myapp_production" {
  destination_database = "myapp_test"

  table "users" {
    sample_rate = 0.05
  }

  table "comments" {
    where {
      user_id = users.id
    }
  }
}
```

The dump for this configuration will include 5% of the records from the `users` table and all of the `comments` records which have a matching foreign key to the 5% of dumped users (all of the comments belonging to the subset set of users after sampling). A table that depends on another can have tables depend on itself, and so on. References are tracked and tables are dumped in the appropriate order.

### Rule configuration

A table block may declare `rule` blocks to add and config behavior for modifying column data before it is written to the dump.

The following rules are currently supported:

#### Mask

The `mask` rule masks data by replacing characters that match the `pattern` regular expression with a `surrogate` character. The default pattern is `[^\s]` and the default surrogate is `*`.

```hcl
database "myapp_production" {
  destination_database = "myapp_test"
  table "users" {
    rule "mask" {
      columns = [pin]
    }
  }
}
```

The dump for this configuration will mask every character in the `pin` column on the `users` table with `*`. You can use `pattern` and `surrogate` to configure this rule differently.

#### Redact

The `redact` rule removes data by removing all characters from character fields or replacing the data with a "nil" value.

```hcl
database "myapp_production" {
  destination_database = "myapp_test"
  table "users" {
    rule "redact" {
      columns = [dob]
    }
  }
}
```

The dump for this configuration will remove data in the `dob` column on the `users`.

### Functions

Some functions are available for use in the HCL configuration file.

#### `now`

`now` can be used to return a `RFC3339` timestamp from the moment the dump started. Please note that it does not return a newly-generated timestamp when called.

```hcl
database "example" {
  table "posts" {
    where = "created_at > '${now()}' - interval 1 day"
  }
}
```

## Limitations

* Related records are obtained via subqueries, nested when necessary, and can perform poorly in many cases.
* The SQL parsing and restoration can fail to write columns with quotes correctly in some cases.

## @TODO:

- enhance sampling with CTE and window function where supported (mysql >=8)
- Replace (Faker?)
- Tokenize
- Bucketing
- Date Shifting
- Time extraction
- support more dialects
