DROP TABLE IF EXISTS `users`;

CREATE TABLE `users` (
  `id` mediumint(8) unsigned NOT NULL auto_increment,
  `email` varchar(255) default NULL,
  `pin` varchar(4),
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1;

INSERT INTO `users` (`email`,`pin`) VALUES ("dis.parturient.montes@google.couk",2756);
INSERT INTO `users` (`email`,`pin`) VALUES ("ut.dolor@protonmail.ca",1146);
INSERT INTO `users` (`email`,`pin`) VALUES ("proin.nisl@protonmail.edu",4780);
INSERT INTO `users` (`email`,`pin`) VALUES ("phasellus@icloud.couk",5113);
INSERT INTO `users` (`email`,`pin`) VALUES ("nunc.quis@icloud.org",9973);
