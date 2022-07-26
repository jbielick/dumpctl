USE `myapp_test`;
DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` mediumint(8) unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `phone` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `setup_complete` tinyint(1) DEFAULT '0',
  `pin` varchar(4) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
);
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `users` VALUES (1,'Ryder Mckenzie','(741) 321-8821','a.ultricies.adipiscing@yahoo.edu',1,'');
INSERT INTO `users` VALUES (3,'Brenda Padilla','(888) 464-1200','purus@google.org',1,'');
INSERT INTO `users` VALUES (4,'Yetta Bryant','(863) 568-6868','mauris.magna.duis@aol.couk',1,'');
INSERT INTO `users` VALUES (5,'Demetria Benton','(185) 627-3418','cursus.integer.mollis@hotmail.com',1,'');
USE `myapp_test`;
DROP TABLE IF EXISTS `appointments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `appointments` (
  `id` mediumint(8) unsigned NOT NULL,
  `notes` text COLLATE utf8mb4_unicode_ci,
  `duration` mediumint(9) DEFAULT NULL,
  `user_id` mediumint(8) unsigned NOT NULL,
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `appointments` VALUES (1,'sed dolor. Fusce mi lorem, vehicula et, rutrum eu, ultrices',5,1,'2022-07-26 21:35:25');
INSERT INTO `appointments` VALUES (2,'eget varius ultrices, mauris ipsum porta elit, a feugiat tellus',0,1,'2022-07-24 21:35:25');
INSERT INTO `appointments` VALUES (4,'eget laoreet posuere, enim nisl elementum purus, accumsan interdum libero',8,4,'2022-07-24 21:35:25');
INSERT INTO `appointments` VALUES (5,'placerat eget, venenatis a, magna. Lorem ipsum dolor sit amet,',7,5,'2022-07-24 21:35:25');
USE `myapp_test`;
DROP TABLE IF EXISTS `comments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `comments` (
  `id` mediumint(8) unsigned NOT NULL,
  `text` text COLLATE utf8mb4_unicode_ci,
  `appointment_id` mediumint(8) unsigned NOT NULL,
  PRIMARY KEY (`id`)
);
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `comments` VALUES (1,'sed dolor. Fusce mi lorem, vehicula et, rutrum eu, ultrices',1);
INSERT INTO `comments` VALUES (2,'eget varius ultrices, mauris ipsum porta elit, a feugiat tellus',1);
INSERT INTO `comments` VALUES (3,'Nulla eget metus eu erat semper rutrum. Fusce dolor quam,',2);
INSERT INTO `comments` VALUES (4,'eget laoreet posuere, enim nisl elementum purus, accumsan interdum libero',4);
INSERT INTO `comments` VALUES (5,'placerat eget, venenatis a, magna. Lorem ipsum dolor sit amet,',5);
