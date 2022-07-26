DROP TABLE IF EXISTS `users`;

CREATE TABLE `users` (
  `id` mediumint(8) unsigned NOT NULL auto_increment,
  `name` varchar(255) default NULL,
  `phone` varchar(100) default NULL,
  `email` varchar(255) default NULL,
  `setup_complete` tinyint(1) default 0,
  `pin` varchar(4),
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1;

INSERT INTO `users` (`name`,`phone`,`email`,`setup_complete`,`pin`)
VALUES
  ("Ryder Mckenzie","(741) 321-8821","a.ultricies.adipiscing@yahoo.edu",1,7400),
  ("Dennis Salas","1-837-288-1215","nulla.eu.neque@protonmail.com",0,1189),
  ("Brenda Padilla","(888) 464-1200","purus@google.org",1,3672),
  ("Yetta Bryant","(863) 568-6868","mauris.magna.duis@aol.couk",1,4470),
  ("Demetria Benton","(185) 627-3418","cursus.integer.mollis@hotmail.com",1,2819);


DROP TABLE IF EXISTS `appointments`;

CREATE TABLE `appointments` (
  `id` mediumint(8) unsigned NOT NULL auto_increment,
  `notes` TEXT default NULL,
  `duration` mediumint default NULL,
  `user_id` mediumint(8) unsigned NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1;

INSERT INTO `appointments` (`notes`,`duration`,`user_id`,`created_at`)
VALUES
  ("sed dolor. Fusce mi lorem, vehicula et, rutrum eu, ultrices",5,1,now()),
  ("eget varius ultrices, mauris ipsum porta elit, a feugiat tellus",0,1,now()-interval 2 day),
  ("Nulla eget metus eu erat semper rutrum. Fusce dolor quam,",6,2,now()-interval 10 day),
  ("eget laoreet posuere, enim nisl elementum purus, accumsan interdum libero",8,4,now()-interval 2 day),
  ("placerat eget, venenatis a, magna. Lorem ipsum dolor sit amet,",7,5,now()-interval 2 day);

DROP TABLE IF EXISTS `comments`;

CREATE TABLE `comments` (
  `id` mediumint(8) unsigned NOT NULL auto_increment,
  `text` TEXT default NULL,
  `appointment_id` mediumint(8) unsigned NOT NULL,
  PRIMARY KEY (`id`)
) AUTO_INCREMENT=1;

INSERT INTO `comments` (`text`,`appointment_id`)
VALUES
  ("sed dolor. Fusce mi lorem, vehicula et, rutrum eu, ultrices",1),
  ("eget varius ultrices, mauris ipsum porta elit, a feugiat tellus",1),
  ("Nulla eget metus eu erat semper rutrum. Fusce dolor quam,",2),
  ("eget laoreet posuere, enim nisl elementum purus, accumsan interdum libero",4),
  ("placerat eget, venenatis a, magna. Lorem ipsum dolor sit amet,",5);
