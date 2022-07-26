USE `myapp_test`;
DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` mediumint(8) unsigned NOT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `pin` varchar(4) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
);
/*!40101 SET character_set_client = @saved_cs_client */;
INSERT INTO `users` VALUES (1,'dis.parturient.montes@google.couk','****');
INSERT INTO `users` VALUES (2,'ut.dolor@protonmail.ca','****');
INSERT INTO `users` VALUES (3,'proin.nisl@protonmail.edu','****');
INSERT INTO `users` VALUES (4,'phasellus@icloud.couk','****');
INSERT INTO `users` VALUES (5,'nunc.quis@icloud.org','****');
