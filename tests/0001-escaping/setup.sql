create table `test` (
  `col` json
);
INSERT INTO `test` (`col`) VALUES
('\"--- !ruby/hash:ActiveSupport::HashWithIndifferentAccess\\nkey1: value1\\nkey2: value2\\n\"');
